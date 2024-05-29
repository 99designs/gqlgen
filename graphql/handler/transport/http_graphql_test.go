package transport_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestGRAPHQL(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.GRAPHQL{})

	t.Run("success", func(t *testing.T) {
		resp := doGraphqlRequest(h, "POST", "/graphql", `{ name }`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("success even if url encoded", func(t *testing.T) {
		resp := doGraphqlRequest(h, "POST", "/graphql", `%7B%20name%20%7D`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		resp := doGraphqlRequest(h, "POST", "/graphql", `{"!"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.Equal(t, `{"errors":[{"message":"Expected Name, found String","locations":[{"line":1,"column":3}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("parse query failure", func(t *testing.T) {
		resp := doGraphqlRequest(h, "POST", "/graphql", `%7B%H7U6Z`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.Equal(t, `{"errors":[{"message":"could not cleanup body: invalid URL escape \"%H7\""}],"data":null}`, resp.Body.String())
	})

	t.Run("validation failure", func(t *testing.T) {
		resp := doGraphqlRequest(h, "POST", "/graphql", `{ title }`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.Equal(t, `{"errors":[{"message":"Cannot query field \"title\" on type \"Query\".","locations":[{"line":1,"column":3}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("execution failure", func(t *testing.T) {
		resp := doGraphqlRequest(h, "POST", "/graphql", `mutation { name }`)
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
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
			"application/graphql",
			"application/graphql; charset=utf-8",
		}

		for _, contentType := range validContentTypes {
			t.Run(fmt.Sprintf("allow for content type %s", contentType), func(t *testing.T) {
				resp := doReq(h, "POST", "/graphql", `{ name }`, contentType)
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
				assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
			})
		}

		invalidContentTypes := []string{
			"",
			"text/plain",
		}

		for _, tc := range invalidContentTypes {
			t.Run(fmt.Sprintf("reject for content type %s", tc), func(t *testing.T) {
				resp := doReq(h, "POST", "/graphql", `{"query":"{ name }"}`, tc)
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
				assert.Equal(t, fmt.Sprintf(`{"errors":[{"message":"%s"}],"data":null}`, "transport not supported"), resp.Body.String())
			})
		}
	})
}

func doGraphqlRequest(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/graphql")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
