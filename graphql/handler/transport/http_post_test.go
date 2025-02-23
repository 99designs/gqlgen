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

func TestPOST(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})

	jsonH := testserver.New()
	jsonH.AddTransport(transport.POST{
		ResponseHeaders: map[string][]string{
			"Content-Type": {"application/json"},
		},
	})

	t.Run("success with accept application/json", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`, "application/json", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("success with accept application/graphql-response+json", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`, "application/graphql-response+json; charset=utf-8", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("success with accept wildcard", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`, "*/*", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("success with json only", func(t *testing.T) {
		resp := doRequest(jsonH, "POST", "/graphql", `{"query":"{ name }"}`, "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", "notjson", "application/json", "application/json")
		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"json request body could not be decoded: invalid character 'o' in literal null (expecting 'u') body:notjson"}],"data":null}`, resp.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "!"}`, "", "application/json")
		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("parse failure with json only", func(t *testing.T) {
		resp := doRequest(jsonH, "POST", "/graphql", `{"query": "!"}`, "", "application/json")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("validation failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "{ title }"}`, "", "application/json")
		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"Cannot query field \"title\" on type \"Query\".","locations":[{"line":1,"column":3}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("validation failure with json only", func(t *testing.T) {
		resp := doRequest(jsonH, "POST", "/graphql", `{"query": "{ title }"}`, "", "application/json")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"Cannot query field \"title\" on type \"Query\".","locations":[{"line":1,"column":3}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("invalid variable", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "query($id:Int!){find(id:$id)}","variables":{"id":false}}`, "", "application/json")
		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"cannot use bool as Int","path":["variable","id"],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("invalid variable with json only", func(t *testing.T) {
		resp := doRequest(jsonH, "POST", "/graphql", `{"query": "query($id:Int!){find(id:$id)}","variables":{"id":false}}`, "", "application/json")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"cannot use bool as Int","path":["variable","id"],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("execution failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query": "mutation { name }"}`, "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, "application/graphql-response+json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"mutations are not supported"}],"data":null}`, resp.Body.String())
	})

	t.Run("execution failure with json only", func(t *testing.T) {
		resp := doRequest(jsonH, "POST", "/graphql", `{"query": "mutation { name }"}`, "", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"mutations are not supported"}],"data":null}`, resp.Body.String())
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
				resp := doReq(h, "POST", "/graphql", `{"query":"{ name }"}`, contentType)
				assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
				assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
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
				assert.JSONEq(t, fmt.Sprintf(`{"errors":[{"message":"%s"}],"data":null}`, "transport not supported"), resp.Body.String())
			})
		}
	})

	t.Run("validate SSE", func(t *testing.T) {
		doReq := func(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
			r := httptest.NewRequest(method, target, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "text/event-stream")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)
			return w
		}

		resp := doReq(h, "POST", "/graphql", `{"query":"{ name }"}`)
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})
}

func doRequest(handler http.Handler, method, target, body, accept, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if accept != "" {
		r.Header.Set("Accept", accept)
	}
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
