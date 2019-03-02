package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"
)

func TestHandlerPOST(t *testing.T) {
	h := GraphQL(&executableSchemaStub{})

	t.Run("success", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ me { name } }"}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("query caching", func(t *testing.T) {
		// Run enough unique queries to evict a bunch of them
		for i := 0; i < 2000; i++ {
			query := `{"query":"` + strings.Repeat(" ", i) + "{ me { name } }" + `"}`
			resp := doRequest(h, "POST", "/graphql", query)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		}

		t.Run("evicted queries run", func(t *testing.T) {
			query := `{"query":"` + strings.Repeat(" ", 0) + "{ me { name } }" + `"}`
			resp := doRequest(h, "POST", "/graphql", query)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		})

		t.Run("non-evicted queries run", func(t *testing.T) {
			query := `{"query":"` + strings.Repeat(" ", 1999) + "{ me { name } }" + `"}`
			resp := doRequest(h, "POST", "/graphql", query)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		})
	})

	t.Run("decode failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", "notjson")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Equal(t, resp.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, `{"errors":[{"message":"json body could not be decoded: invalid character 'o' in literal null (expecting 'u')"}],"data":null}`, resp.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "!"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, resp.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, `{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}]}],"data":null}`, resp.Body.String())
	})

	t.Run("validation failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "{ me { title }}"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, resp.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, `{"errors":[{"message":"Cannot query field \"title\" on type \"User\".","locations":[{"line":1,"column":8}]}],"data":null}`, resp.Body.String())
	})

	t.Run("invalid variable", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "query($id:Int!){user(id:$id){name}}","variables":{"id":false}}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, resp.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, `{"errors":[{"message":"cannot use bool as Int","path":["variable","id"]}],"data":null}`, resp.Body.String())
	})

	t.Run("execution failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "mutation { me { name } }"}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, resp.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, `{"errors":[{"message":"mutations are not supported"}],"data":null}`, resp.Body.String())
	})
}

func TestHandlerGET(t *testing.T) {
	h := GraphQL(&executableSchemaStub{})

	t.Run("success", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query={me{name}}", ``)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=me{id}&variables=notjson", "")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"variables could not be decoded"}],"data":null}`, resp.Body.String())
	})

	t.Run("invalid variable", func(t *testing.T) {
		resp := doRequest(h, "GET", `/graphql?query=query($id:Int!){user(id:$id){name}}&variables={"id":false}`, "")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"cannot use bool as Int","path":["variable","id"]}],"data":null}`, resp.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=!", "")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}]}],"data":null}`, resp.Body.String())
	})

	t.Run("no mutations", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=mutation{me{name}}", "")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"GET requests only allow query operations"}],"data":null}`, resp.Body.String())
	})
}

func TestFileUpload(t *testing.T) {
	t.Run("valid single file upload", func(t *testing.T) {
		stub := &executableSchemaStub{
			MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
				require.Equal(t, len(op.VariableDefinitions), 1)
				require.Equal(t, op.VariableDefinitions[0].Variable, "file")
				return &graphql.Response{Data: []byte(`{"name":"test"}`)}
			},
		}
		handler := GraphQL(stub)

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { fileName, size } }", "variables": { "file": null } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.file"] }`)
		require.NoError(t, err)
		w, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("test"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/graphql", bodyBuf)
		require.NoError(t, err)

		contentType := bodyWriter.FormDataContentType()
		req.Header.Set("Content-Type", contentType)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("valid single file upload with payload", func(t *testing.T) {
		stub := &executableSchemaStub{}
		stub.MutationFunc = func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			require.Equal(t, len(op.VariableDefinitions), 1)
			require.Equal(t, op.VariableDefinitions[0].Variable, "file")
			return &graphql.Response{Data: []byte(`{"name":"test"}`)}
		}
		handler := GraphQL(stub)

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { fileName, size } }", "variables": { "file": null } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.file"] }`)
		require.NoError(t, err)
		w, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("test"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/graphql", bodyBuf)
		require.NoError(t, err)

		contentType := bodyWriter.FormDataContentType()
		req.Header.Set("Content-Type", contentType)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("valid file list upload", func(t *testing.T) {
		stub := &executableSchemaStub{}
		stub.MutationFunc = func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			require.Equal(t, len(op.VariableDefinitions), 1)
			require.Equal(t, op.VariableDefinitions[0].Variable, "files")
			return &graphql.Response{Data: []byte(`{"name":"test"}`)}
		}
		handler := GraphQL(stub)

		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		err := bodyWriter.WriteField("operations", `{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { fileName } }", "variables": { "files": [null, null] } }`)
		require.NoError(t, err)
		err = bodyWriter.WriteField("map", `{ "0": ["variables.files.0"], "1": ["variables.files.1"] }`)
		require.NoError(t, err)
		w0, err := bodyWriter.CreateFormFile("0", "a.txt")
		require.NoError(t, err)
		_, err = w0.Write([]byte("test"))
		require.NoError(t, err)
		w1, err := bodyWriter.CreateFormFile("1", "b.txt")
		require.NoError(t, err)
		_, err = w1.Write([]byte("test"))
		require.NoError(t, err)
		err = bodyWriter.Close()
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/graphql", bodyBuf)
		require.NoError(t, err)

		req.Header.Set("Content-Type", bodyWriter.FormDataContentType())
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

}

func TestHandlerOptions(t *testing.T) {
	h := GraphQL(&executableSchemaStub{})

	resp := doRequest(h, "OPTIONS", "/graphql?query={me{name}}", ``)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "OPTIONS, GET, POST", resp.Header().Get("Allow"))
}

func TestHandlerHead(t *testing.T) {
	h := GraphQL(&executableSchemaStub{})

	resp := doRequest(h, "HEAD", "/graphql?query={me{name}}", ``)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.Code)
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
