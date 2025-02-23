package transport_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestHeadersWithPOST(t *testing.T) {
	t.Run("Headers not set", func(t *testing.T) {
		h := testserver.New()
		h.AddTransport(transport.POST{})

		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`, "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 1)
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
	})

	t.Run("Headers set", func(t *testing.T) {
		headers := map[string][]string{
			"Content-Type": {"application/json; charset: utf8"},
			"Other-Header": {"dummy-post", "another-one"},
		}

		h := testserver.New()
		h.AddTransport(transport.POST{ResponseHeaders: headers})

		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`, "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 2)
		assert.Equal(t, "application/json; charset: utf8", resp.Header().Get("Content-Type"))
		assert.Equal(t, "dummy-post", resp.Header().Get("Other-Header"))
		assert.Equal(t, "another-one", resp.Header().Values("Other-Header")[1])
	})
}

func TestHeadersWithGET(t *testing.T) {
	t.Run("Headers not set", func(t *testing.T) {
		h := testserver.New()
		h.AddTransport(transport.GET{})

		resp := doRequest(h, "GET", "/graphql?query={name}", "", "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 1)
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
	})

	t.Run("Headers set", func(t *testing.T) {
		headers := map[string][]string{
			"Content-Type": {"application/json; charset: utf8"},
			"Other-Header": {"dummy-get"},
		}

		h := testserver.New()
		h.AddTransport(transport.GET{ResponseHeaders: headers})

		resp := doRequest(h, "GET", "/graphql?query={name}", "", "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 2)
		assert.Equal(t, "application/json; charset: utf8", resp.Header().Get("Content-Type"))
		assert.Equal(t, "dummy-get", resp.Header().Get("Other-Header"))
	})
}

func TestHeadersWithGRAPHQL(t *testing.T) {
	t.Run("Headers not set", func(t *testing.T) {
		h := testserver.New()
		h.AddTransport(transport.GRAPHQL{})

		resp := doRequest(h, "POST", "/graphql", `{ name }`, "", "application/graphql")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 1)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	})

	t.Run("Headers set", func(t *testing.T) {
		headers := map[string][]string{
			"Content-Type": {"application/json; charset: utf8"},
			"Other-Header": {"dummy-get-qraphql"},
		}

		h := testserver.New()
		h.AddTransport(transport.GRAPHQL{ResponseHeaders: headers})

		resp := doRequest(h, "POST", "/graphql", `{ name }`, "", "application/graphql")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 2)
		assert.Equal(t, "application/json; charset: utf8", resp.Header().Get("Content-Type"))
		assert.Equal(t, "dummy-get-qraphql", resp.Header().Get("Other-Header"))
	})
}

func TestHeadersWithFormUrlEncoded(t *testing.T) {
	t.Run("Headers not set", func(t *testing.T) {
		h := testserver.New()
		h.AddTransport(transport.UrlEncodedForm{})

		resp := doRequest(h, "POST", "/graphql", `{ name }`, "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 1)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	})

	t.Run("Headers set", func(t *testing.T) {
		headers := map[string][]string{
			"Content-Type": {"application/json; charset: utf8"},
			"Other-Header": {"dummy-get-urlencoded-form"},
		}

		h := testserver.New()
		h.AddTransport(transport.UrlEncodedForm{ResponseHeaders: headers})

		resp := doRequest(h, "POST", "/graphql", `{ name }`, "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Len(t, resp.Header(), 2)
		assert.Equal(t, "application/json; charset: utf8", resp.Header().Get("Content-Type"))
		assert.Equal(t, "dummy-get-urlencoded-form", resp.Header().Get("Other-Header"))
	})
}

func TestHeadersWithMULTIPART(t *testing.T) {
	t.Run("Headers not set", func(t *testing.T) {
		es := &graphql.ExecutableSchemaMock{
			ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
				return graphql.OneShot(graphql.ErrorResponse(ctx, "not implemented"))
			},
			SchemaFunc: func() *ast.Schema {
				return gqlparser.MustLoadSchema(&ast.Source{Input: `
					type Mutation {
						singleUpload(file: Upload!): String!
					}
					scalar Upload
				`})
			},
		}

		h := handler.New(es)
		h.AddTransport(transport.MultipartForm{})

		es.ExecFunc = func(ctx context.Context) graphql.ResponseHandler {
			return graphql.OneShot(&graphql.Response{Data: []byte(`{"singleUpload":"test"}`)})
		}

		operations := `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) }", "variables": { "file": null } }`
		mapData := `{ "0": ["variables.file"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test1",
				contentType: "text/plain",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		resp := httptest.NewRecorder()
		h.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Len(t, resp.Header(), 1)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	})

	t.Run("Headers set", func(t *testing.T) {
		es := &graphql.ExecutableSchemaMock{
			ExecFunc: func(ctx context.Context) graphql.ResponseHandler {
				return graphql.OneShot(graphql.ErrorResponse(ctx, "not implemented"))
			},
			SchemaFunc: func() *ast.Schema {
				return gqlparser.MustLoadSchema(&ast.Source{Input: `
					type Mutation {
						singleUpload(file: Upload!): String!
					}
					scalar Upload
				`})
			},
		}

		h := handler.New(es)
		headers := map[string][]string{
			"Content-Type": {"application/json; charset: utf8"},
			"Other-Header": {"dummy-multipart"},
		}
		h.AddTransport(transport.MultipartForm{ResponseHeaders: headers})

		es.ExecFunc = func(ctx context.Context) graphql.ResponseHandler {
			return graphql.OneShot(&graphql.Response{Data: []byte(`{"singleUpload":"test"}`)})
		}

		operations := `{ "query": "mutation ($file: Upload!) { singleUpload(file: $file) }", "variables": { "file": null } }`
		mapData := `{ "0": ["variables.file"] }`
		files := []file{
			{
				mapKey:      "0",
				name:        "a.txt",
				content:     "test1",
				contentType: "text/plain",
			},
		}
		req := createUploadRequest(t, operations, mapData, files)

		resp := httptest.NewRecorder()
		h.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Len(t, resp.Header(), 2)
		assert.Equal(t, "application/json; charset: utf8", resp.Header().Get("Content-Type"))
		assert.Equal(t, "dummy-multipart", resp.Header().Get("Other-Header"))
	})
}
