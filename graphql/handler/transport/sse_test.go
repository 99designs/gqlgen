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
	initialize := func(sse transport.SSE) *testserver.TestServer {
		h := testserver.New()
		h.AddTransport(sse)
		return h
	}

	initializeWithServer := func(sse transport.SSE) (*testserver.TestServer, *httptest.Server) {
		h := initialize(sse)
		return h, httptest.NewServer(h)
	}

	createHTTPTestRequest := func(query string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	createHTTPRequest := func(url string, query string) (*http.Request, error) {
		req, err := http.NewRequest("POST", url, strings.NewReader(query))
		assert.NoError(t, err, "Request threw error -> %s", err)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req, err
	}

	readLine := func(br *bufio.Reader) string {
		bs, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		return bs
	}

	t.Run("stream failure", func(t *testing.T) {
		h := initialize(transport.SSE{})
		req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{"query":"subscription { name }"}`))
		req.Header.Set("content-type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %s", w.Code)
		assert.Equal(t, `{"errors":[{"message":"transport not supported"}],"data":null}`, w.Body.String())
	})

	t.Run("decode failure", func(t *testing.T) {
		h := initialize(transport.SSE{})
		req := createHTTPTestRequest("notjson")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code, "Request return wrong status -> %s", w.Code)
		assert.Equal(t, `{"errors":[{"message":"json request body could not be decoded: invalid character 'o' in literal null (expecting 'u') body:notjson"}],"data":null}`, w.Body.String())
	})

	t.Run("parse failure", func(t *testing.T) {
		h := initialize(transport.SSE{})
		req := createHTTPTestRequest(`{"query":"subscription {{ name }"}`)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		assert.Equal(t, 422, w.Code, "Request return wrong status -> %s", w.Code)
	})

	t.Run("subscribe", func(t *testing.T) {
		handler, srv := initializeWithServer(transport.SSE{})
		defer srv.Close()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendNextSubscriptionMessage()
		}()

		var Client = &http.Client{}
		req, err := createHTTPRequest(srv.URL, `{"query":"subscription { name }"}`)
		require.NoError(t, err, "Create request threw error -> %s", err)
		res, err := Client.Do(req)
		require.NoError(t, err, "Request threw error -> %s", err)
		defer res.Body.Close()
		assert.Equal(t, 200, res.StatusCode, "Request return wrong status -> %s", res.Status)
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
