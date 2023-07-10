package apollofederatedtracingv1_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1"
	"github.com/99designs/gqlgen/graphql/handler/apollofederatedtracingv1/generated"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"google.golang.org/protobuf/proto"
)

type alwaysError struct{}

func (a *alwaysError) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestApolloTracing(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(&apollofederatedtracingv1.Tracer{})
	h.Use(&delayMiddleware{})

	resp := doRequest(h, http.MethodPost, "/graphql", `{"query":"{ name }"}`)
	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var respData struct {
		Extensions struct {
			FTV1 string `json:"ftv1"`
		} `json:"extensions"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respData))

	tracing := respData.Extensions.FTV1
	pbuf, err := base64.StdEncoding.DecodeString(tracing)
	require.Nil(t, err)

	ftv1 := &generated.Trace{}
	err = proto.Unmarshal(pbuf, ftv1)
	require.Nil(t, err)

	require.NotZero(t, ftv1.StartTime.Nanos)
	require.Less(t, ftv1.StartTime.Nanos, ftv1.EndTime.Nanos)
	require.EqualValues(t, ftv1.EndTime.Nanos-ftv1.StartTime.Nanos, ftv1.DurationNs)

	fmt.Printf("%#v\n", resp.Body.String())
	require.Equal(t, "Query", ftv1.Root.Child[0].ParentType)
	require.Equal(t, "name", ftv1.Root.Child[0].GetResponseName())
	require.Equal(t, "String!", ftv1.Root.Child[0].Type)
}

func TestApolloTracing_Concurrent(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(&apollofederatedtracingv1.Tracer{})
	for i := 0; i < 2; i++ {
		go func() {
			resp := doRequest(h, http.MethodPost, "/graphql", `{"query":"{ name }"}`)
			assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
			var respData struct {
				Extensions struct {
					FTV1 string `json:"ftv1"`
				} `json:"extensions"`
			}
			require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respData))

			tracing := respData.Extensions.FTV1
			pbuf, err := base64.StdEncoding.DecodeString(tracing)
			require.Nil(t, err)

			ftv1 := &generated.Trace{}
			err = proto.Unmarshal(pbuf, ftv1)
			require.Nil(t, err)
			require.NotZero(t, ftv1.StartTime.Nanos)
		}()
	}
}

func TestApolloTracing_withFail(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(extension.AutomaticPersistedQuery{Cache: lru.New(100)})
	h.Use(&apollofederatedtracingv1.Tracer{})

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

func TestApolloTracing_withUnexpectedEOF(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.POST{})
	h.Use(&apollofederatedtracingv1.Tracer{})

	resp := doRequestWithReader(h, http.MethodPost, "/graphql", &alwaysError{})
	assert.Equal(t, http.StatusOK, resp.Code)
}
func doRequest(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	return doRequestWithReader(handler, method, target, strings.NewReader(body))
}

func doRequestWithReader(handler http.Handler, method string, target string,
	reader io.Reader) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, reader)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("apollo-federation-include-trace", "ftv1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}

type delayMiddleware struct{}

func (*delayMiddleware) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	time.Sleep(time.Millisecond)
	return next(ctx)
}

func (*delayMiddleware) ExtensionName() string {
	return "delay"
}

func (*delayMiddleware) Validate(schema graphql.ExecutableSchema) error {
	return nil
}
