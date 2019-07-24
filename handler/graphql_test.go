package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	lru "github.com/hashicorp/golang-lru"
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

	t.Run("validate content type", func(t *testing.T) {
		doReq := func(handler http.Handler, method string, target string, body string, contentType string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(method, target, strings.NewReader(body))
			if contentType != "" {
				r.Header.Set("Content-Type", contentType)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			return w
		}

		validContentTypes := []string{
			"application/json",
			"application/json; charset=utf-8",
		}

		for _, contentType := range validContentTypes {
			t.Run(fmt.Sprintf("allow for content type %s", contentType), func(t *testing.T) {
				resp := doReq(h, "POST", "/graphql", `{"query":"{ me { name } }"}`, contentType)
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
			})
		}

		invalidContentTypes := []struct{ contentType, expectedError string }{
			{"", "error parsing request Content-Type"},
			{"text/plain", "unsupported Content-Type: text/plain"},

			// These content types are currently not supported, but they are supported by other GraphQL servers, like express-graphql.
			{"application/x-www-form-urlencoded", "unsupported Content-Type: application/x-www-form-urlencoded"},
			{"application/graphql", "unsupported Content-Type: application/graphql"},
		}

		for _, tc := range invalidContentTypes {
			t.Run(fmt.Sprintf("reject for content type %s", tc.contentType), func(t *testing.T) {
				resp := doReq(h, "POST", "/graphql", `{"query":"{ me { name } }"}`, tc.contentType)
				assert.Equal(t, http.StatusBadRequest, resp.Code)
				assert.Equal(t, fmt.Sprintf(`{"errors":[{"message":"%s"}],"data":null}`, tc.expectedError), resp.Body.String())
			})
		}
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

	t.Run("valid file list upload with payload and file reuse", func(t *testing.T) {
		test := func(uploadMaxMemory int64) {
			mock := &executableSchemaMock{
				MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
					require.Equal(t, len(op.VariableDefinitions), 1)
					require.Equal(t, op.VariableDefinitions[0].Variable, "req")
					return &graphql.Response{Data: []byte(`{"multipleUploadWithPayload":[{"id":1},{"id":2}]}`)}
				},
			}
			maxMemory := UploadMaxMemory(uploadMaxMemory)
			handler := GraphQL(mock, maxMemory)

			operations := `{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }`
			mapData := `{ "0": ["variables.req.0.file", "variables.req.1.file"] }`
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
			require.Equal(t, `{"data":{"multipleUploadWithPayload":[{"id":1},{"id":2}]}}`, resp.Body.String())
		}

		t.Run("payload smaller than UploadMaxMemory, stored in memory", func(t *testing.T) {
			test(5000)
		})

		t.Run("payload bigger than UploadMaxMemory, persisted to disk", func(t *testing.T) {
			test(2)
		})
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

	cleanUp := func(t *testing.T, closers []io.Closer, tmpFiles []string) {
		for i := len(closers) - 1; 0 <= i; i-- {
			err := closers[i].Close()
			require.Nil(t, err)
		}
		for _, tmpFiles := range tmpFiles {
			err := os.Remove(tmpFiles)
			require.Nil(t, err)
		}
	}

	t.Run("fail to parse multipart", func(t *testing.T) {
		req := &http.Request{
			Method: "POST",
			Header: http.Header{"Content-Type": {`multipart/form-data; boundary="foo123"`}},
			Body:   ioutil.NopCloser(new(bytes.Buffer)),
		}
		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, DefaultUploadMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "failed to parse multipart form")
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("fail parse operation", func(t *testing.T) {
		operations := `invalid operation`
		req := createUploadRequest(t, operations, validMap, validFiles)

		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, DefaultUploadMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "operations form field could not be decoded")
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("fail parse map", func(t *testing.T) {
		mapData := `invalid map`
		req := createUploadRequest(t, validOperations, mapData, validFiles)

		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, DefaultUploadMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "map form field could not be decoded")
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("fail missing file", func(t *testing.T) {
		var files []file
		req := createUploadRequest(t, validOperations, validMap, files)

		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, DefaultUploadMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "failed to get key 0 from form")
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("fail map entry with invalid operations paths prefix", func(t *testing.T) {
		mapData := `{ "0": ["var.file"] }`
		req := createUploadRequest(t, validOperations, mapData, validFiles)

		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, DefaultUploadMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "invalid operations paths for key 0")
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("fail parse request big body", func(t *testing.T) {
		req := createUploadRequest(t, validOperations, validMap, validFiles)

		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		var smallMaxSize int64 = 2
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, smallMaxSize, DefaultUploadMaxMemory)
		require.NotNil(t, err)
		require.Equal(t, err.Error(), "failed to parse multipart form, request body too large")
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("valid request", func(t *testing.T) {
		req := createUploadRequest(t, validOperations, validMap, validFiles)

		var reqParams params
		var closers []io.Closer
		var tmpFiles []string
		w := httptest.NewRecorder()
		err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, DefaultUploadMaxMemory)
		require.Nil(t, err)
		require.Equal(t, "mutation ($file: Upload!) { singleUpload(file: $file) { id } }", reqParams.Query)
		require.Equal(t, "", reqParams.OperationName)
		require.Equal(t, 1, len(reqParams.Variables))
		require.NotNil(t, reqParams.Variables["file"])
		reqParamsFile, ok := reqParams.Variables["file"].(graphql.Upload)
		require.True(t, ok)
		require.Equal(t, "a.txt", reqParamsFile.Filename)
		require.Equal(t, int64(len("test1")), reqParamsFile.Size)
		content, err := ioutil.ReadAll(reqParamsFile.File)
		require.Nil(t, err)
		require.Equal(t, "test1", string(content))
		cleanUp(t, closers, tmpFiles)
	})

	t.Run("valid file list upload with payload and file reuse", func(t *testing.T) {
		operations := `{ "query": "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id } }", "variables": { "req": [ { "id": 1, "file": null }, { "id": 2, "file": null } ] } }`
		mapData := `{ "0": ["variables.req.0.file", "variables.req.1.file"] }`
		files := []file{
			{
				mapKey:  "0",
				name:    "a.txt",
				content: "test1",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		test := func(uploadMaxMemory int64) {
			var reqParams params
			var closers []io.Closer
			var tmpFiles []string
			w := httptest.NewRecorder()
			err := processMultipart(w, req, &reqParams, &closers, &tmpFiles, DefaultUploadMaxSize, uploadMaxMemory)
			require.Nil(t, err)
			require.Equal(t, "mutation($req: [UploadFile!]!) { multipleUploadWithPayload(req: $req) { id } }", reqParams.Query)
			require.Equal(t, "", reqParams.OperationName)
			require.Equal(t, 1, len(reqParams.Variables))
			require.NotNil(t, reqParams.Variables["req"])
			reqParamsFile, ok := reqParams.Variables["req"].([]interface{})
			require.True(t, ok)
			require.Equal(t, 2, len(reqParamsFile))
			for i, item := range reqParamsFile {
				itemMap := item.(map[string]interface{})
				require.Equal(t, fmt.Sprint(itemMap["id"]), fmt.Sprint(i+1))
				file := itemMap["file"].(graphql.Upload)
				require.Equal(t, "a.txt", file.Filename)
				require.Equal(t, int64(len("test1")), file.Size)
				require.Nil(t, err)
				content, err := ioutil.ReadAll(file.File)
				require.Nil(t, err)
				require.Equal(t, "test1", string(content))
			}
			cleanUp(t, closers, tmpFiles)
		}

		t.Run("payload smaller than UploadMaxMemory, stored in memory", func(t *testing.T) {
			test(5000)
		})

		t.Run("payload bigger than UploadMaxMemory, persisted to disk", func(t *testing.T) {
			test(2)
		})
	})
}

func TestAddUploadToOperations(t *testing.T) {
	key := "0"

	t.Run("fail missing all variables", func(t *testing.T) {
		file, _ := os.Open("path/to/file")
		request := &params{}

		upload := graphql.Upload{
			File:     file,
			Filename: "a.txt",
			Size:     int64(5),
		}
		path := "variables.req.0.file"
		err := addUploadToOperations(request, upload, key, path)
		require.NotNil(t, err)
		require.Equal(t, "path is missing \"variables.\" prefix, key: 0, path: variables.req.0.file", err.Error())
	})

	t.Run("valid variable", func(t *testing.T) {
		file, _ := os.Open("path/to/file")
		request := &params{
			Variables: map[string]interface{}{
				"file": nil,
			},
		}

		upload := graphql.Upload{
			File:     file,
			Filename: "a.txt",
			Size:     int64(5),
		}

		expected := &params{
			Variables: map[string]interface{}{
				"file": upload,
			},
		}

		path := "variables.file"
		err := addUploadToOperations(request, upload, key, path)
		require.Nil(t, err)

		require.Equal(t, request, expected)
	})

	t.Run("valid nested variable", func(t *testing.T) {
		file, _ := os.Open("path/to/file")
		request := &params{
			Variables: map[string]interface{}{
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

		expected := &params{
			Variables: map[string]interface{}{
				"req": []interface{}{
					map[string]interface{}{
						"file": upload,
					},
				},
			},
		}

		path := "variables.req.0.file"
		err := addUploadToOperations(request, upload, key, path)
		require.Nil(t, err)

		require.Equal(t, request, expected)
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
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}

func TestBytesRead(t *testing.T) {
	t.Run("test concurrency", func(t *testing.T) {
		// Test for the race detector, to verify a Read that doesn't yield any bytes
		// is okay to use from multiple goroutines. This was our historic behavior.
		// See golang.org/issue/7856
		r := bytesReader{s: &([]byte{})}
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				var buf [1]byte
				r.Read(buf[:])
			}()
			go func() {
				defer wg.Done()
				r.Read(nil)
			}()
		}
		wg.Wait()
	})

	t.Run("fail to read if pointer is nil", func(t *testing.T) {
		n, err := (&bytesReader{}).Read(nil)
		require.Equal(t, 0, n)
		require.NotNil(t, err)
		require.Equal(t, "byte slice pointer is nil", err.Error())
	})

	t.Run("read using buffer", func(t *testing.T) {
		data := []byte("0123456789")
		r := bytesReader{s: &data}

		got := make([]byte, 0, 11)
		buf := make([]byte, 1)
		for {
			n, err := r.Read(buf)
			if n < 0 {
				require.Fail(t, "unexpected bytes read size")
			}
			got = append(got, buf[:n]...)
			if err != nil {
				if err == io.EOF {
					break
				}
				require.Fail(t, "unexpected error while reading", err.Error())
			}
		}
		require.Equal(t, "0123456789", string(got))
	})

	t.Run("read updated pointer value", func(t *testing.T) {
		data := []byte("0123456789")
		pointer := &data
		r := bytesReader{s: pointer}
		data[2] = []byte("9")[0]

		got := make([]byte, 0, 11)
		buf := make([]byte, 1)
		for {
			n, err := r.Read(buf)
			if n < 0 {
				require.Fail(t, "unexpected bytes read size")
			}
			got = append(got, buf[:n]...)
			if err != nil {
				if err == io.EOF {
					break
				}
				require.Fail(t, "unexpected error while reading", err.Error())
			}
		}
		require.Equal(t, "0193456789", string(got))
	})
}

type memoryPersistedQueryCache struct {
	cache *lru.Cache
}

func newMemoryPersistedQueryCache(size int) (*memoryPersistedQueryCache, error) {
	cache, err := lru.New(size)
	return &memoryPersistedQueryCache{cache: cache}, err
}

func (c *memoryPersistedQueryCache) Add(ctx context.Context, hash string, query string) {
	c.cache.Add(hash, query)
}

func (c *memoryPersistedQueryCache) Get(ctx context.Context, hash string) (string, bool) {
	val, ok := c.cache.Get(hash)
	if !ok {
		return "", ok
	}
	return val.(string), ok
}
func TestAutomaticPersistedQuery(t *testing.T) {
	cache, err := newMemoryPersistedQueryCache(1000)
	require.NoError(t, err)
	h := GraphQL(&executableSchemaStub{}, EnablePersistedQueryCache(cache))
	t.Run("automatic persisted query POST", func(t *testing.T) {
		// normal queries should be unaffected
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ me { name } }"}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())

		// first pass: optimistic hash without query string
		resp = doRequest(h, "POST", "/graphql", `{"extensions":{"persistedQuery":{"sha256Hash":"b8d9506e34c83b0e53c2aa463624fcea354713bc38f95276e6f0bd893ffb5b88","version":1}}}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"PersistedQueryNotFound"}],"data":null}`, resp.Body.String())
		// second pass: query with query string and query hash
		resp = doRequest(h, "POST", "/graphql", `{"query":"{ me { name } }", "extensions":{"persistedQuery":{"sha256Hash":"b8d9506e34c83b0e53c2aa463624fcea354713bc38f95276e6f0bd893ffb5b88","version":1}}}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		// future requests without query string
		resp = doRequest(h, "POST", "/graphql", `{"extensions":{"persistedQuery":{"sha256Hash":"b8d9506e34c83b0e53c2aa463624fcea354713bc38f95276e6f0bd893ffb5b88","version":1}}}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("automatic persisted query GET", func(t *testing.T) {
		// normal queries should be unaffected
		resp := doRequest(h, "GET", "/graphql?query={me{name}}", "")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())

		// first pass: optimistic hash without query string
		resp = doRequest(h, "GET", `/graphql?extensions={"persistedQuery":{"version":1,"sha256Hash":"b58723c4fd7ce18043ae53635b304ba6cee765a67009645b04ca01e80ce1c065"}}`, "")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"PersistedQueryNotFound"}],"data":null}`, resp.Body.String())
		// second pass: query with query string and query hash
		resp = doRequest(h, "GET", `/graphql?query={me{name}}&extensions={"persistedQuery":{"sha256Hash":"b58723c4fd7ce18043ae53635b304ba6cee765a67009645b04ca01e80ce1c065","version":1}}}`, "")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
		// future requests without query string
		resp = doRequest(h, "GET", `/graphql?extensions={"persistedQuery":{"version":1,"sha256Hash":"b58723c4fd7ce18043ae53635b304ba6cee765a67009645b04ca01e80ce1c065"}}`, "")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})
}
