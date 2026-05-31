package transport_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestSSE(t *testing.T) {
	pingInterval := time.Second * 1

	initialize := func() *testserver.TestServer {
		h := testserver.New()
		h.AddTransport(transport.SSE{})
		return h
	}

	initializeWithServer := func() (*testserver.TestServer, *httptest.Server) {
		h := initialize()
		return h, httptest.NewServer(h)
	}

	initializeKeepAliveWithServer := func() (*testserver.TestServer, *httptest.Server) {
		h := testserver.New()
		h.AddTransport(transport.SSE{
			KeepAlivePingInterval: pingInterval,
		})
		return h, httptest.NewServer(h)
	}

	createHTTPTestRequest := func(query string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	createHTTPRequest := func(url string, query string) *http.Request {
		req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(query))
		require.NoError(t, err, "Request threw error -> %s", err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	readLine := func(br *bufio.Reader) string {
		bs, err := br.ReadString('\n')
		require.NoError(t, err)
		return bs
	}

	t.Run("stream failure", func(t *testing.T) {
		h := initialize()
		req := httptest.NewRequest(
			http.MethodPost,
			"/graphql",
			strings.NewReader(`{"query":"subscription { name }"}`),
		)
		req.Header.Set("content-type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %d", w.Code)
		assert.JSONEq(
			t,
			`{"errors":[{"message":"transport not supported"}],"data":null}`,
			w.Body.String(),
		)
	})

	t.Run("fail on null body", func(t *testing.T) {
		h := initialize()
		req := createHTTPTestRequest("null")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		br := bufio.NewReader(w.Body)

		assert.Equal(t, ":\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(
			t,
			`data: {"errors":[{"message":"no operation provided","extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`+"\n",
			readLine(br),
		)
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err := br.ReadByte()
		assert.Equal(t, err, io.EOF)
	})

	t.Run("decode failure", func(t *testing.T) {
		h := initialize()
		req := createHTTPTestRequest("notjson")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %d", w.Code)
		assert.JSONEq(
			t,
			`{"errors":[{"message":"json request body could not be decoded: invalid character 'o' in literal null (expecting 'u') body:notjson"}],"data":null}`,
			w.Body.String(),
		)
	})

	t.Run("parse failure", func(t *testing.T) {
		h := initialize()
		req := createHTTPTestRequest(`{"query":"subscription {{ name }"}`)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		br := bufio.NewReader(w.Body)

		assert.Equal(t, ":\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(
			t,
			"data: {\"errors\":[{\"message\":\"Expected Name, found {\",\"locations\":[{\"line\":1,\"column\":15}],\"extensions\":{\"code\":\"GRAPHQL_PARSE_FAILED\"}}],\"data\":null}\n",
			readLine(br),
		)
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err := br.ReadByte()
		assert.Equal(t, err, io.EOF)
	})

	t.Run("subscribe", func(t *testing.T) {
		handler, srv := initializeWithServer()
		defer srv.Close()

		var wg sync.WaitGroup
		wg.Go(func() {
			handler.SendNextSubscriptionMessage()
		})

		client := &http.Client{}
		req := createHTTPRequest(srv.URL, `{"query":"subscription { name }"}`)
		res, err := client.Do(req)
		require.NoError(t, err, "Request threw error -> %s", err)
		defer func() {
			require.NoError(t, res.Body.Close())
		}()

		assert.Equal(t, 200, res.StatusCode, "Request return wrong status -> %d", res.Status)
		assert.Equal(t, "keep-alive", res.Header.Get("Connection"))
		assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))

		br := bufio.NewReader(res.Body)

		assert.Equal(t, ":\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(t, "data: {\"data\":{\"name\":\"test\"}}\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		wg.Go(func() {
			handler.SendNextSubscriptionMessage()
		})

		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(t, "data: {\"data\":{\"name\":\"test\"}}\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		wg.Go(func() {
			handler.SendCompleteSubscriptionMessage()
		})

		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err = br.ReadByte()
		assert.Equal(t, err, io.EOF)

		wg.Wait()
	})

	t.Run("subscribe with keep alive", func(t *testing.T) {
		handler, srv := initializeKeepAliveWithServer()
		defer srv.Close()

		var wg sync.WaitGroup
		wg.Go(func() {
			// Wait for ping interval to trigger
			time.Sleep(pingInterval + time.Millisecond*100)
		})

		client := &http.Client{}
		req := createHTTPRequest(srv.URL, `{"query":"subscription { name }"}`)
		res, err := client.Do(req)
		require.NoError(t, err, "Request threw error -> %s", err)
		defer func() {
			require.NoError(t, res.Body.Close())
		}()

		assert.Equal(t, 200, res.StatusCode, "Request return wrong status -> %d", res.Status)
		assert.Equal(t, "keep-alive", res.Header.Get("Connection"))
		assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))

		br := bufio.NewReader(res.Body)

		assert.Equal(t, ":\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))
		assert.Equal(t, ": ping\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		wg.Go(func() {
			handler.SendCompleteSubscriptionMessage()
		})

		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err = br.ReadByte()
		assert.Equal(t, err, io.EOF)

		wg.Wait()
	})

	t.Run("min event interval paces rapid events", func(t *testing.T) {
		interval := 10 * time.Millisecond
		responses := []*graphql.Response{
			{Data: []byte(`{"name":"test1"}`)},
			{Data: []byte(`{"name":"test2"}`)},
			{Data: []byte(`{"name":"test3"}`)},
		}

		req := createHTTPTestRequest(`{"query":"subscription { name }"}`)
		w := httptest.NewRecorder()

		start := time.Now()
		transport.SSE{MinEventInterval: interval}.Do(
			w,
			req,
			&sseGraphExecutor{responses: responses},
		)
		elapsed := time.Since(start)

		assert.GreaterOrEqual(t, elapsed, interval*time.Duration(len(responses)))
		assert.Equal(t, 200, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Equal(t, len(responses), strings.Count(body, "event: next\n"))
		assert.Contains(t, body, `data: {"data":{"name":"test1"}}`)
		assert.Contains(t, body, `data: {"data":{"name":"test2"}}`)
		assert.Contains(t, body, `data: {"data":{"name":"test3"}}`)
		assert.Contains(t, body, "event: complete\n")
	})
}

type sseGraphExecutor struct {
	responses []*graphql.Response
}

func (e *sseGraphExecutor) CreateOperationContext(
	context.Context,
	*graphql.RawParams,
) (*graphql.OperationContext, gqlerror.List) {
	return &graphql.OperationContext{}, nil
}

func (e *sseGraphExecutor) DispatchOperation(
	ctx context.Context,
	_ *graphql.OperationContext,
) (graphql.ResponseHandler, context.Context) {
	index := 0
	return func(context.Context) *graphql.Response {
		if index >= len(e.responses) {
			return nil
		}

		resp := e.responses[index]
		index++
		return resp
	}, ctx
}

func (e *sseGraphExecutor) DispatchError(_ context.Context, errs gqlerror.List) *graphql.Response {
	return &graphql.Response{Errors: errs}
}
