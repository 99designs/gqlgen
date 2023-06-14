package transport_test

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestSSE(t *testing.T) {
	initialize := func() *testserver.TestServer {
		h := testserver.New()
		h.AddTransport(transport.SSE{})
		return h
	}

	initializeWithServer := func() (*testserver.TestServer, *httptest.Server) {
		h := initialize()
		return h, httptest.NewServer(h)
	}

	createHTTPTestRequest := func(query string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	createHTTPRequest := func(url string, query string) *http.Request {
		req, err := http.NewRequest("POST", url, strings.NewReader(query))
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
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{"query":"subscription { name }"}`))
		req.Header.Set("content-type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, `{"errors":[{"message":"transport not supported"}],"data":null}`, w.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		h := initialize()
		req := createHTTPTestRequest("notjson")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %d", w.Code)
		assert.Equal(t, `{"errors":[{"message":"json request body could not be decoded: invalid character 'o' in literal null (expecting 'u') body:notjson"}],"data":null}`, w.Body.String())
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
		assert.Equal(t, "data: {\"errors\":[{\"message\":\"Expected Name, found {\",\"locations\":[{\"line\":1,\"column\":15}],\"extensions\":{\"code\":\"GRAPHQL_PARSE_FAILED\"}}],\"data\":null}\n", readLine(br))
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendNextSubscriptionMessage()
		}()

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

		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendNextSubscriptionMessage()
		}()

		assert.Equal(t, "event: next\n", readLine(br))
		assert.Equal(t, "data: {\"data\":{\"name\":\"test\"}}\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendCompleteSubscriptionMessage()
		}()

		assert.Equal(t, "event: complete\n", readLine(br))
		assert.Equal(t, "\n", readLine(br))

		_, err = br.ReadByte()
		assert.Equal(t, err, io.EOF)

		wg.Wait()
	})
}
