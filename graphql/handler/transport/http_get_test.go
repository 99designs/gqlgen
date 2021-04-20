package transport_test

import (
	"net/http"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/cache"

	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
)

func TestGET(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.GET{})

	t.Run("success", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query={name}", ``)
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("has json content-type header", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query={name}", ``)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	})

	t.Run("decode failure", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query={name}&variables=notjson", "")
		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Equal(t, `{"errors":[{"message":"variables could not be decoded"}],"data":null}`, resp.Body.String())
	})

	t.Run("invalid variable", func(t *testing.T) {
		resp := doRequest(h, "GET", `/graphql?query=query($id:Int!){find(id:$id)}&variables={"id":false}`, "")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, `{"errors":[{"message":"cannot use bool as Int","path":["variable","id"],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=!", "")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, `{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("no mutations", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=mutation{name}", "")
		assert.Equal(t, http.StatusNotAcceptable, resp.Code, resp.Body.String())
		assert.Equal(t, `{"errors":[{"message":"GET requests only allow query operations"}],"data":null}`, resp.Body.String())
	})

	t.Run("cache control", func(t *testing.T) {
		t.Run("write extensions", func(t *testing.T) {
			hWithCache := testserver.NewCache()
			hWithCache.AddTransport(transport.GET{
				EnableCache: true,
			})
			hWithCache.Use(cache.Extension{})

			resp := doRequest(hWithCache, "GET", "/graphql?query={name}", "")
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"},"extensions":{"cacheControl":{"version":1,"hints":[{"path":["name"],"maxAge":10,"scope":"PUBLIC"}]}}}`, resp.Body.String())
		})

		t.Run("write cache control header", func(t *testing.T) {
			hWithCache := testserver.NewCache()
			hWithCache.AddTransport(transport.GET{
				EnableCache: true,
			})
			hWithCache.Use(cache.Extension{})

			resp := doRequest(hWithCache, "GET", "/graphql?query={name}", "")
			assert.Equal(t, "max-age: 10 public", resp.Header().Get("Cache-Control"))
		})

		t.Run("not writes cache control header", func(t *testing.T) {
			hWithCache := testserver.NewCache()
			hWithCache.AddTransport(transport.GET{})
			hWithCache.Use(cache.Extension{})

			resp := doRequest(hWithCache, "GET", "/graphql?query={name}", "")
			assert.Empty(t, resp.Header().Get("Cache-Control"))
		})

	})
}
