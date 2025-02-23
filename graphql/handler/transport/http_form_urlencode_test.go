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

func TestUrlEncodedForm(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.UrlEncodedForm{})

	t.Run("success json", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`, "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("success urlencoded", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `query=%7B%20name%20%7D`, "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("success plain", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `query={ name }`, "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("decode failure json", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", "notjson", "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"Unexpected Name \"notjson\"","locations":[{"line":1,"column":1}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("decode failure urlencoded", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", "query=%7Bnot-good", "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"Expected Name, found \u003cInvalid\u003e","locations":[{"line":1,"column":6}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`, resp.Body.String())
	})

	t.Run("parse query failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":{"wrong": "format"}}`, "", "application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"errors":[{"message":"could not cleanup body: json: cannot unmarshal object into Go struct field RawParams.query of type string"}],"data":null}`, resp.Body.String())
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
			"application/x-www-form-urlencoded",
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
}
