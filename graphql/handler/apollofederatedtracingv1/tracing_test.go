package apollofederatedtracingv1_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1"
	"github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1/generated"
	"github.com/99designs/gqlgen/graphql/handler/apollotracing"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/protobuf/proto"
)

func TestApolloTracing(t *testing.T) {
	now := time.Unix(0, 0)

	graphql.Now = func() time.Time {
		defer func() {
			now = now.Add(100 * time.Nanosecond)
		}()
		return now
	}

	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(&apollofederatedtracingv1.Tracer{})

	resp := doRequest(h, http.MethodPost, "/graphql", `{"query":"{ name }"}`)
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var respData struct {
		Extensions struct {
			FTV1 string `json:"ftv1"`
		} `json:"extensions"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respData))

	tracing := respData.Extensions.FTV1
	pbuf, _ := base64.StdEncoding.DecodeString(tracing)
	ftv1 := &generated.Trace{}
	err := proto.Unmarshal(pbuf, ftv1)
	require.Nil(t, err)

	require.Zero(t, ftv1.StartTime.Nanos, ftv1.StartTime.Nanos)
	require.EqualValues(t, 900, ftv1.EndTime.Nanos)
	require.EqualValues(t, 900, ftv1.DurationNs)

	fmt.Printf("%#v\n", resp.Body.String())
	require.Equal(t, "Query", ftv1.Root.Child[0].ParentType)
	require.Equal(t, "name", ftv1.Root.Child[0].GetResponseName())
	require.Equal(t, "String!", ftv1.Root.Child[0].Type)
}

func TestApolloTracing_withFail(t *testing.T) {
	now := time.Unix(0, 0)

	graphql.Now = func() time.Time {
		defer func() {
			now = now.Add(100 * time.Nanosecond)
		}()
		return now
	}

	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(extension.AutomaticPersistedQuery{Cache: lru.New(100)})
	h.Use(apollotracing.Tracer{})

	resp := doRequest(h, http.MethodPost, "/graphql", `{"operationName":"A","extensions":{"persistedQuery":{"version":1,"sha256Hash":"338bbc16ac780daf81845339fbf0342061c1e9d2b702c96d3958a13a557083a6"}}}`)
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	b := resp.Body.Bytes()
	t.Log(string(b))
	var respData struct {
		Errors gqlerror.List
	}
	require.NoError(t, json.Unmarshal(b, &respData))
	require.Len(t, respData.Errors, 1)
	require.Equal(t, "PersistedQueryNotFound", respData.Errors[0].Message)
}

func doRequest(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("apollo-federation-include-trace", "ftv1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
