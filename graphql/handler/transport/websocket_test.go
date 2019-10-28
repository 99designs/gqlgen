package transport_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

func TestWebsocket(t *testing.T) {
	next := make(chan struct{})
	handler := newServer(next)
	handler.AddTransport(transport.WebsocketTransport{})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	t.Run("client must send valid json", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		writeRaw(c, "hello")

		msg := readOp(c)
		assert.Equal(t, "connection_error", msg.Type)
		assert.Equal(t, `{"message":"invalid json"}`, string(msg.Payload))
	})

	t.Run("client can terminate before init", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionTerminateMsg}))

		_, _, err := c.ReadMessage()
		assert.Equal(t, websocket.CloseNormalClosure, err.(*websocket.CloseError).Code)
	})

	t.Run("client must send init first", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: startMsg}))

		msg := readOp(c)
		assert.Equal(t, connectionErrorMsg, msg.Type)
		assert.Equal(t, `{"message":"unexpected message start"}`, string(msg.Payload))
	})

	t.Run("server acks init", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))

		assert.Equal(t, connectionAckMsg, readOp(c).Type)
		assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)
	})

	t.Run("client can terminate before run", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		assert.Equal(t, connectionAckMsg, readOp(c).Type)
		assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionTerminateMsg}))

		_, _, err := c.ReadMessage()
		assert.Equal(t, websocket.CloseNormalClosure, err.(*websocket.CloseError).Code)
	})

	t.Run("client gets parse errors", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		assert.Equal(t, connectionAckMsg, readOp(c).Type)
		assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{
			Type:    startMsg,
			ID:      "test_1",
			Payload: json.RawMessage(`{"query": "!"}`),
		}))

		msg := readOp(c)
		assert.Equal(t, errorMsg, msg.Type)
		assert.Equal(t, `{"errors":[{"message":"Unexpected !","locations":[{"line":1,"column":1}]}],"data":null}`, string(msg.Payload))
	})

	t.Run("client can receive data", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		assert.Equal(t, connectionAckMsg, readOp(c).Type)
		assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{
			Type:    startMsg,
			ID:      "test_1",
			Payload: json.RawMessage(`{"query": "subscription { user { title } }"}`),
		}))

		next <- struct{}{}
		msg := readOp(c)
		assert.Equal(t, dataMsg, msg.Type)
		assert.Equal(t, "test_1", msg.ID)
		assert.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		next <- struct{}{}
		msg = readOp(c)
		assert.Equal(t, dataMsg, msg.Type)
		assert.Equal(t, "test_1", msg.ID)
		assert.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		require.NoError(t, c.WriteJSON(&operationMessage{Type: stopMsg, ID: "test_1"}))

		msg = readOp(c)
		assert.Equal(t, completeMsg, msg.Type)
		assert.Equal(t, "test_1", msg.ID)
	})
}

func TestWebsocketWithKeepAlive(t *testing.T) {
	next := make(chan struct{})
	h := newServer(next)
	h.AddTransport(transport.WebsocketTransport{
		KeepAlivePingInterval: 10 * time.Millisecond,
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	c := wsConnect(srv.URL)
	defer c.Close()

	require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
	assert.Equal(t, connectionAckMsg, readOp(c).Type)
	assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)

	require.NoError(t, c.WriteJSON(&operationMessage{
		Type:    startMsg,
		ID:      "test_1",
		Payload: json.RawMessage(`{"query": "subscription { user { title } }"}`),
	}))

	// keepalive
	msg := readOp(c)
	assert.Equal(t, connectionKeepAliveMsg, msg.Type)

	// server message
	next <- struct{}{}
	msg = readOp(c)
	assert.Equal(t, dataMsg, msg.Type)

	// keepalive
	msg = readOp(c)
	assert.Equal(t, connectionKeepAliveMsg, msg.Type)
}

func TestWebsocketInitFunc(t *testing.T) {
	next := make(chan struct{})

	t.Run("accept connection if WebsocketInitFunc is NOT provided", func(t *testing.T) {
		h := newServer(next)
		h.AddTransport(transport.WebsocketTransport{})
		srv := httptest.NewServer(h)
		defer srv.Close()

		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))

		assert.Equal(t, connectionAckMsg, readOp(c).Type)
		assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)
	})

	t.Run("accept connection if WebsocketInitFunc is provided and is accepting connection", func(t *testing.T) {
		h := newServer(next)
		h.AddTransport(transport.WebsocketTransport{
			InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, error) {
				return context.WithValue(ctx, "newkey", "newvalue"), nil
			},
		})
		srv := httptest.NewServer(h)
		defer srv.Close()

		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))

		assert.Equal(t, connectionAckMsg, readOp(c).Type)
		assert.Equal(t, connectionKeepAliveMsg, readOp(c).Type)
	})

	t.Run("reject connection if WebsocketInitFunc is provided and is accepting connection", func(t *testing.T) {
		h := newServer(next)
		h.AddTransport(transport.WebsocketTransport{
			InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, error) {
				return ctx, errors.New("invalid init payload")
			},
		})
		srv := httptest.NewServer(h)
		defer srv.Close()

		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))

		msg := readOp(c)
		assert.Equal(t, connectionErrorMsg, msg.Type)
		assert.Equal(t, `{"message":"invalid init payload"}`, string(msg.Payload))
	})

	t.Run("can return context for request from WebsocketInitFunc", func(t *testing.T) {
		es := &graphql.ExecutableSchemaMock{
			QueryFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
				assert.Equal(t, "newvalue", ctx.Value("newkey"))
				return &graphql.Response{Data: []byte(`{"empty":"ok"}`)}
			},
			SchemaFunc: func() *ast.Schema {
				return gqlparser.MustLoadSchema(&ast.Source{Input: `
				schema { query: Query }
				type Query {
					empty: String
				}
			`})
			},
		}
		h := handler.New(es)

		h.AddTransport(transport.WebsocketTransport{
			InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, error) {
				return context.WithValue(ctx, "newkey", "newvalue"), nil
			},
		})

		c := client.New(h)

		socket := c.Websocket("{ empty } ")
		defer socket.Close()
		var resp struct {
			Empty string
		}
		err := socket.Next(&resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp.Empty)
	})
}

func newServer(next chan struct{}) *handler.Server {
	es := &graphql.ExecutableSchemaMock{
		QueryFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return graphql.ErrorResponse(ctx, "queries are not supported")
		},
		MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return graphql.ErrorResponse(ctx, "mutations are not supported")
		},
		SubscriptionFunc: func(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
			return func() *graphql.Response {
				select {
				case <-ctx.Done():
					return nil
				case <-next:
					return &graphql.Response{
						Data: []byte(`{"name":"test"}`),
					}
				}
			}
		},
		SchemaFunc: func() *ast.Schema {
			return gqlparser.MustLoadSchema(&ast.Source{Input: `
				schema { query: Query }
				type Query {
					me: User!
					user(id: Int): User!
				}
				type User { name: String! }
			`})
		},
	}
	return handler.New(es)
}

func wsConnect(url string) *websocket.Conn {
	c, resp, err := websocket.DefaultDialer.Dial(strings.Replace(url, "http://", "ws://", -1), nil)
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	return c
}

func writeRaw(conn *websocket.Conn, msg string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		panic(err)
	}
}

func readOp(conn *websocket.Conn) operationMessage {
	var msg operationMessage
	if err := conn.ReadJSON(&msg); err != nil {
		panic(err)
	}
	return msg
}

// copied out from weboscket.go to keep these private

const (
	connectionInitMsg      = "connection_init"      // Client -> Server
	connectionTerminateMsg = "connection_terminate" // Client -> Server
	startMsg               = "start"                // Client -> Server
	stopMsg                = "stop"                 // Client -> Server
	connectionAckMsg       = "connection_ack"       // Server -> Client
	connectionErrorMsg     = "connection_error"     // Server -> Client
	dataMsg                = "data"                 // Server -> Client
	errorMsg               = "error"                // Server -> Client
	completeMsg            = "complete"             // Server -> Client
	connectionKeepAliveMsg = "ka"                   // Server -> Client
)

type operationMessage struct {
	Payload json.RawMessage `json:"payload,omitempty"`
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
}
