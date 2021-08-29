package main

import (
	"encoding/json"
	"github.com/99designs/gqlgen/graphql/handler/cache"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
)

const query = `{
	  latestPost {
		id
		comments {
		  text
		  post {
			id
		  }
		}
		readByCurrentUser
	  }
	}`

const expectedExtension = `{
    "cacheControl": {
      "version": 1,
      "hints": [
        {
          "path": [
            "latestPost"
          ],
          "maxAge": 10,
          "scope": "PUBLIC"
        },
        {
          "path": [
            "latestPost",
            "comments"
          ],
          "maxAge": 1000,
          "scope": "PUBLIC"
        },
        {
          "path": [
            "latestPost",
            "readByCurrentUser"
          ],
          "maxAge": 2,
          "scope": "PRIVATE"
        },
        {
          "path": [
            "latestPost",
            "comments",
            1,
            "post"
          ],
          "maxAge": 10,
          "scope": "PUBLIC"
        },
        {
          "path": [
            "latestPost",
            "comments",
            0,
            "post"
          ],
          "maxAge": 10,
          "scope": "PUBLIC"
        }
      ]
    }}`

func TestServer(t *testing.T) {
	c := client.New(new())
	actual, err := c.RawPost(query)
	require.NoError(t, err)

	var expected map[string]interface{}
	err = json.Unmarshal([]byte(expectedExtension), &expected)

	require.NoError(t, err)
	require.Nil(t, actual.Errors)
	require.NotNil(t, actual.Data)
	expectedCacheControl := expected["cacheControl"].(map[string]interface{})
	actualCacheControl := actual.Extensions["cacheControl"].(map[string]interface{})
	require.Equal(t, expectedCacheControl["version"], actualCacheControl["version"])
	require.ElementsMatch(t, expectedCacheControl["hints"], actualCacheControl["hints"])
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	return w
}

func TestCacheExtension(t *testing.T) {
	h := new()

	t.Run("GET", func(t *testing.T) {
		t.Run("write extensions", func(t *testing.T) {
			resp := doRequest(cache.Middleware(h), "GET", "/graphql?query={name}", "")
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"},"extensions":{"cacheControl":{"version":1,"hints":[{"path":["name"],"maxAge":10,"scope":"PUBLIC"}]}}}`, resp.Body.String())
		})

		t.Run("write cache control header", func(t *testing.T) {
			resp := doRequest(cache.Middleware(h), "GET", "/graphql?query={name}", "")
			assert.Equal(t, "max-age: 10 public", resp.Header().Get("Cache-Control"))
		})

		t.Run("not writes cache control header", func(t *testing.T) {
			resp := doRequest(h, "GET", "/graphql?query={name}", "")
			assert.Empty(t, resp.Header().Get("Cache-Control"))
		})
	})

	t.Run("POST", func(t *testing.T) {
		t.Run("write extensions", func(t *testing.T) {
			resp := doRequest(cache.Middleware(h), "POST", "/graphql", `{"query":"{ name }"}`)
			assert.Equal(t, http.StatusOK, resp.Code)
			assert.Equal(t, `{"data":{"name":"test"},"extensions":{"cacheControl":{"version":1,"hints":[{"path":["name"],"maxAge":10,"scope":"PUBLIC"}]}}}`, resp.Body.String())
		})

		t.Run("write cache control header", func(t *testing.T) {
			resp := doRequest(cache.Middleware(h), "POST", "/graphql", `{"query":"{ name }"}`)
			assert.Equal(t, "max-age: 10 public", resp.Header().Get("Cache-Control"))
		})

		t.Run("not writes cache control header", func(t *testing.T) {
			resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`)
			assert.Empty(t, resp.Header().Get("Cache-Control"))
		})
	})
}
