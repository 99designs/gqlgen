package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vektah/gqlgen/neelance/errors"

	"github.com/stretchr/testify/assert"
)

func TestHandlerPOST(t *testing.T) {
	h := GraphQL(func(ctx context.Context, document string, operationName string, variables map[string]interface{}, w io.Writer) []*errors.QueryError {
		if document == "error" {
			return []*errors.QueryError{errors.Errorf("error!")}
		}

		w.Write([]byte(`{"data":{}}`))
		return nil
	})

	t.Run("ok", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"me{id}"}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{}}`, resp.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", "notjson")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"json body could not be decoded"}]}`, resp.Body.String())
	})

	t.Run("execution failure", func(t *testing.T) {
		resp := doRequest(h, "POST", "/graphql", `{"query":"error"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"error!"}]}`, resp.Body.String())
	})
}

func TestHandlerGET(t *testing.T) {
	h := GraphQL(func(ctx context.Context, document string, operationName string, variables map[string]interface{}, w io.Writer) []*errors.QueryError {
		if document == "error" {
			return []*errors.QueryError{errors.Errorf("error!")}
		}

		w.Write([]byte(`{"data":{}}`))
		return nil
	})

	t.Run("just query", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=me{id}", ``)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{}}`, resp.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=me{id}&variables=notjson", "")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"variables could not be decoded"}]}`, resp.Body.String())
	})

	t.Run("execution failure", func(t *testing.T) {
		resp := doRequest(h, "GET", "/graphql?query=error", "")
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"error!"}]}`, resp.Body.String())
	})
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
