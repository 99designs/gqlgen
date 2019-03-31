package handler

import (
	"bytes"
	"context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHandlerComplexity(t *testing.T) {
	t.Run("static complexity", func(t *testing.T) {
		h := GraphQL(&executableSchemaStub{}, ComplexityLimit(2))

		t.Run("below complexity limit", func(t *testing.T) {
			resp := doRequest(h, "POST", "/graphql", `{"query":"{ me { name } }"}`)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		})

		t.Run("above complexity limit", func(t *testing.T) {
			resp := doRequest(h, "POST", "/graphql", `{"query":"{ a: me { name } b: me { name } }"}`)
			assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
			assert.Equal(t, `{"errors":[{"message":"operation has complexity 4, which exceeds the limit of 2"}],"data":null}`, resp.Body.String())
		})
	})

	t.Run("dynamic complexity", func(t *testing.T) {
		h := GraphQL(&executableSchemaStub{}, ComplexityLimitFunc(func(ctx context.Context) int {
			reqCtx := graphql.GetRequestContext(ctx)
			if strings.Contains(reqCtx.RawQuery, "dummy") {
				return 4
			}
			return 2
		}))

		t.Run("below complexity limit", func(t *testing.T) {
			resp := doRequest(h, "POST", "/graphql", `{"query":"{ me { name } }"}`)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		})

		t.Run("above complexity limit", func(t *testing.T) {
			resp := doRequest(h, "POST", "/graphql", `{"query":"{ a: me { name } b: me { name } }"}`)
			assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
			assert.Equal(t, `{"errors":[{"message":"operation has complexity 4, which exceeds the limit of 2"}],"data":null}`, resp.Body.String())
		})

		t.Run("within dynamic complexity limit", func(t *testing.T) {
			resp := doRequest(h, "POST", "/graphql", `{"query":"{ a: me { name } dummy: me { name } }"}`)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		})
	})
}

func TestFileUpload(t *testing.T) {

	t.Run("valid single file upload", func(t *testing.T) {
		mock := &executableSchemaMock{
			MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
				require.Equal(t, len(op.VariableDefinitions), 1)
				require.Equal(t, op.VariableDefinitions[0].Variable, "file")
				return &graphql.Response{Data: []byte(`{"singleUpload":{"id":1}}`)}
			},
		}
		handler := GraphQL(mock)

		operations := `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id } }", "variables": { "file": null } }`
		mapData := `{ "0": ["variables.file"] }`
		files := []file{
			{
				mapKey:  "0",
				name:    "a.txt",
				content: "test1",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"singleUpload":{"id":1}}}`, resp.Body.String())
	})

	t.Run("valid single file upload with payload", func(t *testing.T) {
		mock := &executableSchemaMock{
			MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
				require.Equal(t, len(op.VariableDefinitions), 1)
				require.Equal(t, op.VariableDefinitions[0].Variable, "req")
				return &graphql.Response{Data: []byte(`{"singleUploadWithPayload":{"id":1}}`)}
			},
		}
		handler := GraphQL(mock)

		operations := `{ "query": "mutation ($req: UploadFile!) { singleUploadWithPayload(req: $req) { id } }", "variables": { "req": {"file": null, "id": 1 } } }`
		mapData := `{ "0": ["variables.req.file"] }`
		files := []file{
			{
				mapKey:  "0",
				name:    "a.txt",
				content: "test1",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"singleUploadWithPayload":{"id":1}}}`, resp.Body.String())
	})

	t.Run("valid file list upload", func(t *testing.T) {
		mock := &executableSchemaMock{
			MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
				require.Equal(t, len(op.VariableDefinitions), 1)
				require.Equal(t, op.VariableDefinitions[0].Variable, "files")
				return &graphql.Response{Data: []byte(`{"multipleUpload":[{"id":1},{"id":2}]}`)}
			},
		}
		handler := GraphQL(mock)

		operations := `{ "query": "mutation($files: [Upload!]!) { multipleUpload(files: $files) { id } }", "variables": { "files": [null, null] } }`
		mapData := `{ "0": ["variables.files.0"], "1": ["variables.files.1"] }`
		files := []file{
			{
				mapKey:  "0",
				name:    "a.txt",
				content: "test1",
			},
			{
				mapKey:  "1",
				name:    "b.txt",
				content: "test2",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"multipleUpload":[{"id":1},{"id":2}]}}`, resp.Body.String())
	})

	t.Run("valid file list upload with payload", func(t *testing.T) {
		mock := &executableSchemaMock{
			MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
				require.Equal(t, len(op.VariableDefinitions), 1)
				require.Equal(t, op.VariableDefinitions[0].Variable, "req")
				return &graphql.Response{Data: []byte(`{"multipleUploadWithPayload":[{"id":1},{"id":2}]}`)}
			},
		}
		handler := GraphQL(mock)

		operations := `{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }`
		mapData := `{ "0": ["variables.req.0.file"], "1": ["variables.req.1.file"] }`
		files := []file{
			{
				mapKey:  "0",
				name:    "a.txt",
				content: "test1",
			},
			{
				mapKey:  "1",
				name:    "b.txt",
				content: "test2",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, `{"data":{"multipleUploadWithPayload":[{"id":1},{"id":2}]}}`, resp.Body.String())
	})
}

func TestProcessMultipart(t *testing.T) {
	validOperations := `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) { id } }", "variables": { "file": null } }`
	validMap := `{ "0": ["variables.file"] }`
	validFiles := []file{
		{
			mapKey:  "0",
			name:    "a.txt",
			content: "test1",
		},
	}

	t.Run("fail to parse multipart", func(t *testing.T) {
		req := &http.Request{
			Method: "POST",
			Header: http.Header{"Content-Type": {`multipart/form-data; boundary="foo123"`}},
			Body:   ioutil.NopCloser(new(bytes.Buffer)),
		}
		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.NotNil(t, err)
		errMsg := err.Error()
		require.Equal(t, errMsg, "failed to parse multipart form")
	})

	t.Run("fail parse operation", func(t *testing.T) {
		operations := `invalid operation`
		req := createUploadRequest(t, operations, validMap, validFiles)

		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "operations form field could not be decoded")
	})

	t.Run("fail parse map", func(t *testing.T) {
		mapData := `invalid map`
		req := createUploadRequest(t, validOperations, mapData, validFiles)

		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "map form field could not be decoded")
	})

	t.Run("fail missing file", func(t *testing.T) {
		var files []file
		req := createUploadRequest(t, validOperations, validMap, files)

		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "failed to get key 0 from form")
	})

	t.Run("fail map entry with two values", func(t *testing.T) {
		mapData := `{ "0": ["variables.file", "variables.file"] }`
		req := createUploadRequest(t, validOperations, mapData, validFiles)

		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "invalid value for key 0")
	})

	t.Run("fail map entry with invalid prefix", func(t *testing.T) {
		mapData := `{ "0": ["var.file"] }`
		req := createUploadRequest(t, validOperations, mapData, validFiles)

		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "invalid value for key 0")
	})

	t.Run("valid request", func(t *testing.T) {
		req := createUploadRequest(t, validOperations, validMap, validFiles)

		var reqParams params
		err := processMultipart(req, &reqParams, DefaultFileMaxMemory)
		require.Nil(t, err)
		require.Equal(t, "mutation ($file: Upload!) { singleUpload(file: $file) { id } }", reqParams.Query)
		require.Equal(t, "", reqParams.OperationName)
		require.Equal(t, 1, len(reqParams.Variables))
		require.NotNil(t, reqParams.Variables["file"])
		reqParamsFile, ok := reqParams.Variables["file"].(graphql.Upload)
		require.True(t, ok)
		require.Equal(t, "a.txt", reqParamsFile.Filename)
		require.Equal(t, int64(len("test1")), reqParamsFile.Size)
		var content []byte
		content, err = ioutil.ReadAll(reqParamsFile.File)
		require.Nil(t, err)
		require.Equal(t, "test1", string(content))
	})
}

func TestAddUploadToOperations(t *testing.T) {

	t.Run("fail missing all variables", func(t *testing.T) {
		file, err := os.Open("path/to/file")
		var operations map[string]interface{}
		upload := graphql.Upload{
			File:     file,
			Filename: "a.txt",
			Size:     int64(5),
		}
		path := "variables.req.0.file"
		err = addUploadToOperations(operations, upload, path)
		require.NotNil(t, err)
		require.Equal(t, "variables is missing, path: variables.req.0.file", err.Error())
	})

	t.Run("valid variable", func(t *testing.T) {
		file, err := os.Open("path/to/file")
		operations := map[string]interface{}{
			"variables": map[string]interface{}{
				"file": nil,
			},
		}

		upload := graphql.Upload{
			File:     file,
			Filename: "a.txt",
			Size:     int64(5),
		}

		expected := map[string]interface{}{
			"variables": map[string]interface{}{
				"file": upload,
			},
		}

		path := "variables.file"
		err = addUploadToOperations(operations, upload, path)
		require.Nil(t, err)

		require.Equal(t, operations, expected)
	})

	t.Run("valid nested variable", func(t *testing.T) {
		file, err := os.Open("path/to/file")
		operations := map[string]interface{}{
			"variables": map[string]interface{}{
				"req": []interface{}{
					map[string]interface{}{
						"file": nil,
					},
				},
			},
		}

		upload := graphql.Upload{
			File:     file,
			Filename: "a.txt",
			Size:     int64(5),
		}

		expected := map[string]interface{}{
			"variables": map[string]interface{}{
				"req": []interface{}{
					map[string]interface{}{
						"file": upload,
					},
				},
			},
		}

		path := "variables.req.0.file"
		err = addUploadToOperations(operations, upload, path)
		require.Nil(t, err)

		require.Equal(t, operations, expected)
	})
}

type file struct {
	mapKey  string
	name    string
	content string
}

func createUploadRequest(t *testing.T, operations, mapData string, files []file) *http.Request {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	err := bodyWriter.WriteField("operations", operations)
	require.NoError(t, err)

	err = bodyWriter.WriteField("map", mapData)
	require.NoError(t, err)

	for i := range files {
		ff, err := bodyWriter.CreateFormFile(files[i].mapKey, files[i].name)
		require.NoError(t, err)
		_, err = ff.Write([]byte(files[i].content))
		require.NoError(t, err)
	}
	err = bodyWriter.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/graphql", bodyBuf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", bodyWriter.FormDataContentType())
	return req
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
