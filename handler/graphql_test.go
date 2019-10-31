package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
)

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

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
