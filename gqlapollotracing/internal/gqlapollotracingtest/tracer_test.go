package gqlapollotracingtest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/99designs/gqlgen/gqlapollotracing"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/assert"
)

func TestNewTracer(t *testing.T) {
	h := handler.GraphQL(
		NewExecutableSchema(
			Config{
				Resolvers: NewResolver(),
			},
		),
		handler.RequestMiddleware(gqlapollotracing.RequestMiddleware()),
		handler.Tracer(gqlapollotracing.NewTracer()),
	)

	t.Run("success", func(t *testing.T) {
		var mu sync.Mutex
		now := time.Date(2018, 10, 30, 9, 0, 0, 0, time.UTC)
		gqlapollotracing.SetTimeNowFunc(func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			now = now.Add(100 * time.Millisecond)
			return now
		})
		defer gqlapollotracing.SetTimeNowFunc(time.Now)

		resp := doRequest(h, "POST", "/query", `{"query":"{ todos { id text } }"}`)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"todos":[{"id":"Todo:1","text":"Play with cat"}]},"extensions":{"tracing":{"startTime":"2018-10-30T09:00:00Z","endTime":"2018-10-30T09:00:01Z","duration":1000000000,"parsing":{"startOffset":0,"duration":100000000},"validation":{"startOffset":200000000,"duration":100000000},"execution":{"resolvers":[{"startOffset":400000000,"duration":500000000,"path":["todos"],"parentType":"Query","fieldName":"todos","returnType":"[Todo!]!"},{"startOffset":500000000,"duration":100000000,"path":["todos",0,"id"],"parentType":"Todo","fieldName":"id","returnType":"ID!"},{"startOffset":700000000,"duration":100000000,"path":["todos",0,"text"],"parentType":"Todo","fieldName":"text","returnType":"String!"}]}}}}`, resp.Body.String())
	})
}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
