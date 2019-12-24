package apollotracing_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/apollotracing"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/gqlerror"
)

func TestApolloTracing(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(apollotracing.Tracer{})

	resp := doRequest(h, "POST", "/graphql", `{"query":"{ name }"}`)
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var respData struct {
		Extensions struct {
			Tracing apollotracing.TracingExtension `json:"tracing"`
		} `json:"extensions"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respData))

	tracing := &respData.Extensions.Tracing

	require.EqualValues(t, 1, tracing.Version)

	require.EqualValues(t, 0, tracing.StartTime.UnixNano())
	require.EqualValues(t, 700, tracing.EndTime.UnixNano())
	require.EqualValues(t, 700, tracing.Duration)

	require.EqualValues(t, 100, tracing.Parsing.StartOffset)
	require.EqualValues(t, 100, tracing.Parsing.Duration)

	require.EqualValues(t, 300, tracing.Validation.StartOffset)
	require.EqualValues(t, 100, tracing.Validation.Duration)

	require.EqualValues(t, 500, tracing.Execution.Resolvers[0].StartOffset)
	require.EqualValues(t, 100, tracing.Execution.Resolvers[0].Duration)
	require.EqualValues(t, []interface{}{"name"}, tracing.Execution.Resolvers[0].Path)
	require.EqualValues(t, "Query", tracing.Execution.Resolvers[0].ParentType)
	require.EqualValues(t, "name", tracing.Execution.Resolvers[0].FieldName)
	require.EqualValues(t, "String!", tracing.Execution.Resolvers[0].ReturnType)
}

func TestApolloTracing_withFail(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(extension.AutomaticPersistedQuery{Cache: lru.New(100)})
	h.Use(apollotracing.Tracer{})

	resp := doRequest(h, "POST", "/graphql", `{"operationName":"A","extensions":{"persistedQuery":{"version":1,"sha256Hash":"338bbc16ac780daf81845339fbf0342061c1e9d2b702c96d3958a13a557083a6"}}}`)
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	b := resp.Body.Bytes()
	t.Log(string(b))
	var respData struct {
		Errors gqlerror.List
	}
	require.NoError(t, json.Unmarshal(b, &respData))
	require.Equal(t, 1, len(respData.Errors))
	require.Equal(t, "PersistedQueryNotFound", respData.Errors[0].Message)
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
