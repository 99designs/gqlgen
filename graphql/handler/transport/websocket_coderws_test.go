package transport_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestCoderWebsocketImplementation(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.Websocket{
		Implementation: transport.CoderWebsocketImplementation{},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx := context.Background()
	c := coderwsDial(ctx, t, srv.URL)
	defer c.CloseNow()

	coderwsWriteOperation(ctx, t, c, operationMessage{Type: graphqltransportwsConnectionInitMsg})
	assert.Equal(t, graphqltransportwsConnectionAckMsg, coderwsReadOperation(ctx, t, c).Type)

	coderwsWriteOperation(ctx, t, c, operationMessage{
		Type:    graphqltransportwsSubscribeMsg,
		ID:      "test_1",
		Payload: json.RawMessage(`{"query": "subscription { name }"}`),
	})

	h.SendNextSubscriptionMessage()
	msg := coderwsReadOperation(ctx, t, c)
	require.Equal(t, graphqltransportwsNextMsg, msg.Type, string(msg.Payload))
	require.Equal(t, "test_1", msg.ID, string(msg.Payload))
	require.JSONEq(t, `{"data":{"name":"test"}}`, string(msg.Payload))
}

func TestCoderWebsocketImplementationPreservesAcceptOptionsSubprotocolPreference(t *testing.T) {
	h := testserver.New()
	h.AddTransport(transport.Websocket{
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: coderws.AcceptOptions{
				Subprotocols: []string{graphqltransportwsSubprotocol},
			},
		},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx := context.Background()
	c := coderwsDial(ctx, t, srv.URL, "graphql-ws", graphqltransportwsSubprotocol)
	defer c.CloseNow()

	require.Equal(t, graphqltransportwsSubprotocol, c.Subprotocol())
	coderwsWriteOperation(ctx, t, c, operationMessage{Type: graphqltransportwsConnectionInitMsg})
	assert.Equal(t, graphqltransportwsConnectionAckMsg, coderwsReadOperation(ctx, t, c).Type)
}

func TestCoderWebsocketImplementationEnforcesReadDeadline(t *testing.T) {
	closeFuncCalled := make(chan bool, 1)
	h := testserver.New()
	h.AddTransport(transport.Websocket{
		Implementation:   transport.CoderWebsocketImplementation{},
		MissingPongOk:    false,
		PingPongInterval: 5 * time.Millisecond,
		CloseFunc: func(_ context.Context, _ int) {
			closeFuncCalled <- true
		},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx := context.Background()
	c := coderwsDial(ctx, t, srv.URL)
	defer c.CloseNow()

	coderwsWriteOperation(ctx, t, c, operationMessage{Type: graphqltransportwsConnectionInitMsg})
	assert.Equal(t, graphqltransportwsConnectionAckMsg, coderwsReadOperation(ctx, t, c).Type)
	assert.Equal(t, graphqltransportwsPingMsg, coderwsReadOperation(ctx, t, c).Type)

	select {
	case res := <-closeFuncCalled:
		assert.True(t, res)
	case <-time.NewTimer(30 * time.Millisecond).C:
		assert.Fail(t, "The close handler was not called in time")
	}
}

func coderwsDial(
	ctx context.Context,
	t *testing.T,
	url string,
	subprotocols ...string,
) *coderws.Conn {
	t.Helper()

	if len(subprotocols) == 0 {
		subprotocols = []string{graphqltransportwsSubprotocol}
	}

	wsURL := strings.ReplaceAll(url, "http://", "ws://")
	c, resp, err := coderws.Dial(ctx, wsURL, &coderws.DialOptions{
		Subprotocols: subprotocols,
	})
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}

	return c
}

func coderwsWriteOperation(
	ctx context.Context,
	t *testing.T,
	c *coderws.Conn,
	msg operationMessage,
) {
	t.Helper()

	data, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, c.Write(ctx, coderws.MessageText, data))
}

func coderwsReadOperation(ctx context.Context, t *testing.T, c *coderws.Conn) operationMessage {
	t.Helper()

	messageType, r, err := c.Reader(ctx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, messageType)

	data, err := io.ReadAll(r)
	require.NoError(t, err)

	var msg operationMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	return msg
}
