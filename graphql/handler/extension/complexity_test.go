package extension_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestHandlerComplexity(t *testing.T) {
	h := testserver.New()
	h.Use(extension.ComplexityLimit(func(ctx context.Context, rc *graphql.OperationContext) int {
		if rc.RawQuery == "{ ok: name }" {
			return 4
		}
		return 2
	}))
	h.AddTransport(&transport.POST{})

	t.Run("below complexity limit", func(t *testing.T) {
		h.SetCalculatedComplexity(2)
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		require.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("above complexity limit", func(t *testing.T) {
		h.SetCalculatedComplexity(4)
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`)
		require.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		require.Equal(t, `{"errors":[{"message":"operation has complexity 4, which exceeds the limit of 2"}],"data":null}`, resp.Body.String())
	})

	t.Run("within dynamic complexity limit", func(t *testing.T) {
		h.SetCalculatedComplexity(4)
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ ok: name }"}`)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		require.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})
}

func TestFixedComplexity(t *testing.T) {
	h := testserver.New()
	h.Use(extension.FixedComplexityLimit(2))
	h.AddTransport(&transport.POST{})

	t.Run("below complexity limit", func(t *testing.T) {
		h.SetCalculatedComplexity(2)
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		require.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("above complexity limit", func(t *testing.T) {
		h.SetCalculatedComplexity(4)
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`)
		require.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		require.Equal(t, `{"errors":[{"message":"operation has complexity 4, which exceeds the limit of 2"}],"data":null}`, resp.Body.String())
	})
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
