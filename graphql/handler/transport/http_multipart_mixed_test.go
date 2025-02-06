package transport_test

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestMultipartMixed(t *testing.T) {
	initialize := func() *testserver.TestServer {
		h := testserver.New()
		h.AddTransport(transport.MultipartMixed{
			Boundary: "graphql",
		})
		return h
	}

	initializeWithServer := func() (*testserver.TestServer, *httptest.Server) {
		h := initialize()
		return h, httptest.NewServer(h)
	}

	createHTTPRequest := func(url string, query string) *http.Request {
		req, err := http.NewRequest("POST", url, strings.NewReader(query))
		require.NoError(t, err, "Request threw error -> %s", err)
		req.Header.Set("Accept", "multipart/mixed")
		req.Header.Set("content-type", "application/json; charset=utf-8")
		return req
	}

	doRequest := func(handler http.Handler, target, body string) *httptest.ResponseRecorder {
		r := createHTTPRequest(target, body)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)
		return w
	}

	t.Run("decode failure", func(t *testing.T) {
		handler, srv := initializeWithServer()
		resp := doRequest(handler, srv.URL, "notjson")
		assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(
			t,
			`{"errors":[{"message":"json request body could not be decoded: invalid character 'o' in literal null (expecting 'u') body:notjson"}],"data":null}`,
			resp.Body.String(),
		)
	})

	t.Run("parse failure", func(t *testing.T) {
		handler, srv := initializeWithServer()
		resp := doRequest(handler, srv.URL, `{"query": "!"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(
			t,
			`{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}],"extensions":{"code":"GRAPHQL_PARSE_FAILED"}}],"data":null}`,
			resp.Body.String(),
		)
	})

	t.Run("validation failure", func(t *testing.T) {
		handler, srv := initializeWithServer()
		resp := doRequest(handler, srv.URL, `{"query": "{ title }"}`)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(
			t,
			`{"errors":[{"message":"Cannot query field \"title\" on type \"Query\".","locations":[{"line":1,"column":3}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`,
			resp.Body.String(),
		)
	})

	t.Run("invalid variable", func(t *testing.T) {
		handler, srv := initializeWithServer()
		resp := doRequest(handler, srv.URL,
			`{"query": "query($id:Int!){find(id:$id)}","variables":{"id":false}}`,
		)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.JSONEq(
			t,
			`{"errors":[{"message":"cannot use bool as Int","path":["variable","id"],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`,
			resp.Body.String(),
		)
	})

	readLine := func(br *bufio.Reader) string {
		bs, err := br.ReadString('\n')
		require.NoError(t, err)
		return bs
	}

	t.Run("initial and incremental patches un-aggregated", func(t *testing.T) {
		handler, srv := initializeWithServer()
		defer srv.Close()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendNextSubscriptionMessage()
		}()

		client := &http.Client{}
		req := createHTTPRequest(
			srv.URL,
			`{"query":"query { ... @defer { name } }"}`,
		)
		res, err := client.Do(req)
		require.NoError(t, err, "Request threw error -> %s", err)
		defer func() {
			require.NoError(t, res.Body.Close())
		}()

		assert.Equal(t, 200, res.StatusCode, "Request return wrong status -> %d", res.Status)
		assert.Equal(t, "keep-alive", res.Header.Get("Connection"))
		assert.Contains(t, res.Header.Get("Content-Type"), "multipart/mixed")
		assert.Contains(t, res.Header.Get("Content-Type"), `boundary="graphql"`)

		br := bufio.NewReader(res.Body)

		assert.Equal(t, "--graphql\r\n", readLine(br))
		assert.Equal(t, "Content-Type: application/json\r\n", readLine(br))
		assert.Equal(t, "\r\n", readLine(br))
		assert.JSONEq(t,
			"{\"data\":{\"name\":null},\"hasNext\":true}\r\n",
			readLine(br),
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendNextSubscriptionMessage()
		}()

		assert.Equal(t, "--graphql\r\n", readLine(br))
		assert.Equal(t, "Content-Type: application/json\r\n", readLine(br))
		assert.Equal(t, "\r\n", readLine(br))
		assert.JSONEq(
			t,
			"{\"incremental\":[{\"data\":{\"name\":\"test\"},\"hasNext\":false}],\"hasNext\":false}\r\n",
			readLine(br),
		)

		assert.Equal(t, "--graphql--\r\n", readLine(br))

		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.SendCompleteSubscriptionMessage()
		}()

		_, err = br.ReadByte()
		assert.Equal(t, err, io.EOF)

		wg.Wait()
	})

	t.Run("initial and incremental patches aggregated", func(t *testing.T) {
		handler := testserver.New()
		handler.AddTransport(transport.MultipartMixed{
			Boundary:        "graphql",
			DeliveryTimeout: time.Hour,
		})

		srv := httptest.NewServer(handler)
		defer srv.Close()

		var err error
		var res *http.Response

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{}
			req := createHTTPRequest(
				srv.URL,
				`{"query":"query { ... @defer { name } }"}`,
			)
			res, err = client.Do(req) //nolint:bodyclose // false positive
		}()

		handler.SendNextSubscriptionMessage()
		handler.SendNextSubscriptionMessage()
		handler.SendCompleteSubscriptionMessage()
		wg.Wait()

		require.NoError(t, err, "Request threw error -> %s", err)
		defer func() {
			require.NoError(t, res.Body.Close())
		}()

		assert.Equal(t, 200, res.StatusCode, "Request return wrong status -> %d", res.Status)
		assert.Equal(t, "keep-alive", res.Header.Get("Connection"))
		assert.Contains(t, res.Header.Get("Content-Type"), "multipart/mixed")
		assert.Contains(t, res.Header.Get("Content-Type"), `boundary="graphql"`)

		br := bufio.NewReader(res.Body)
		assert.Equal(t, "--graphql\r\n", readLine(br))
		assert.Equal(t, "Content-Type: application/json\r\n", readLine(br))
		assert.Equal(t, "\r\n", readLine(br))
		assert.JSONEq(t,
			"{\"data\":{\"name\":null},\"hasNext\":true}\r\n",
			readLine(br),
		)

		assert.Equal(t, "--graphql\r\n", readLine(br))
		assert.Equal(t, "Content-Type: application/json\r\n", readLine(br))
		assert.Equal(t, "\r\n", readLine(br))
		assert.JSONEq(
			t,
			"{\"incremental\":[{\"data\":{\"name\":\"test\"},\"hasNext\":false}],\"hasNext\":false}\r\n",
			readLine(br),
		)

		assert.Equal(t, "--graphql--\r\n", readLine(br))

		_, err = br.ReadByte()
		assert.Equal(t, err, io.EOF)
	})
}
