package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWebsocket(t *testing.T) {
	next := make(chan struct{})
	h := GraphQL(&executableSchemaStub{next})

	srv := httptest.NewServer(h)
	defer srv.Close()

	t.Run("client must send valid json", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		writeRaw(c, "hello")

		msg := readOp(c)
		require.Equal(t, connectionErrorMsg, msg.Type)
		require.Equal(t, `{"message":"invalid json"}`, string(msg.Payload))
	})

	t.Run("client can terminate before init", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionTerminateMsg}))

		_, _, err := c.ReadMessage()
		require.Equal(t, websocket.CloseNormalClosure, err.(*websocket.CloseError).Code)
	})

	t.Run("client must send init first", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: startMsg}))

		msg := readOp(c)
		require.Equal(t, connectionErrorMsg, msg.Type)
		require.Equal(t, `{"message":"unexpected message start"}`, string(msg.Payload))
	})

	t.Run("server acks init", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))

		require.Equal(t, connectionAckMsg, readOp(c).Type)
	})

	t.Run("client can terminate before run", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		require.Equal(t, connectionAckMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionTerminateMsg}))

		_, _, err := c.ReadMessage()
		require.Equal(t, websocket.CloseNormalClosure, err.(*websocket.CloseError).Code)
	})

	t.Run("client gets parse errors", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		require.Equal(t, connectionAckMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{
			Type:    startMsg,
			ID:      "test_1",
			Payload: json.RawMessage(`{"query": "!"}`),
		}))

		msg := readOp(c)
		require.Equal(t, errorMsg, msg.Type)
		require.Equal(t, `[{"message":"Unexpected !","locations":[{"line":1,"column":1}]}]`, string(msg.Payload))
	})

	t.Run("client can receive data", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		require.Equal(t, connectionAckMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{
			Type:    startMsg,
			ID:      "test_1",
			Payload: json.RawMessage(`{"query": "subscription { user { title } }"}`),
		}))

		next <- struct{}{}
		msg := readOp(c)
		require.Equal(t, dataMsg, msg.Type)
		require.Equal(t, "test_1", msg.ID)
		require.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		next <- struct{}{}
		msg = readOp(c)
		require.Equal(t, dataMsg, msg.Type)
		require.Equal(t, "test_1", msg.ID)
		require.Equal(t, `{"data":{"name":"test"}}`, string(msg.Payload))

		require.NoError(t, c.WriteJSON(&operationMessage{Type: stopMsg, ID: "test_1"}))

		msg = readOp(c)
		require.Equal(t, completeMsg, msg.Type)
		require.Equal(t, "test_1", msg.ID)
	})
}

func TestWebsocketWithKeepAlive(t *testing.T) {
	next := make(chan struct{})
	h := GraphQL(&executableSchemaStub{next}, WebsocketKeepAliveDuration(10*time.Millisecond))

	srv := httptest.NewServer(h)
	defer srv.Close()

	t.Run("client must receive keepalive", func(t *testing.T) {
		c := wsConnect(srv.URL)
		defer c.Close()

		require.NoError(t, c.WriteJSON(&operationMessage{Type: connectionInitMsg}))
		require.Equal(t, connectionAckMsg, readOp(c).Type)

		require.NoError(t, c.WriteJSON(&operationMessage{
			Type:    startMsg,
			ID:      "test_1",
			Payload: json.RawMessage(`{"query": "subscription { user { title } }"}`),
		}))

		// keepalive
		msg := readOp(c)
		require.Equal(t, connectionKeepAliveMsg, msg.Type)

		// server message
		next <- struct{}{}
		msg = readOp(c)
		require.Equal(t, dataMsg, msg.Type)

		// keepalive
		msg = readOp(c)
		require.Equal(t, connectionKeepAliveMsg, msg.Type)
	})
}

func wsConnect(url string) *websocket.Conn {
	c, _, err := websocket.DefaultDialer.Dial(strings.Replace(url, "http://", "ws://", -1), nil)
	if err != nil {
		panic(err)
	}
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
