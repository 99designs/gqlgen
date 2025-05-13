package transport_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
)

type wrapTestServer struct {
	*testserver.TestServer
}

type wrapWriter struct {
	http.ResponseWriter
}

func (w *wrapWriter) WriteJson(response *graphql.Response) {
	w.Header().Add("my-customized-header", "hello")
	b, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(b)
}

func (s *wrapTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wrapWriter := &wrapWriter{w}
	s.TestServer.ServeHTTP(wrapWriter, r)
}

func TestJSONWriter(t *testing.T) {
	wrapServer := &wrapTestServer{TestServer: testserver.New()}
	wrapServer.AddTransport(transport.GET{})

	t.Run("allows to set the customized header", func(t *testing.T) {
		resp := doRequest(wrapServer, "GET", "/graphql?query={name}", ``, "application/json", "application/json")
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, "hello", resp.Header().Get("my-customized-header"))
		assert.JSONEq(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})
}
