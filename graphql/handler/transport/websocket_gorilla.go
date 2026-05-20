package transport

import (
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/gorilla/websocket"
)

// GorillaWebsocketImplementation adapts github.com/gorilla/websocket to the gqlgen
// websocket transport.
//
// Deprecated: github.com/gorilla/websocket is no longer maintained again. The
// Gorilla-based websocket implementation remains available as the default for
// backwards compatibility. New applications should provide a
// WebsocketImplementation backed by their preferred websocket implementation.
type GorillaWebsocketImplementation = gorillaWebsocketImplementation

type gorillaWebsocketImplementation struct {
	Upgrader websocket.Upgrader
}

func (u gorillaWebsocketImplementation) Accept(
	w http.ResponseWriter,
	r *http.Request,
	options WebsocketAcceptOptions,
) (WebsocketConn, error) {
	upgrader := u.Upgrader
	for _, subprotocol := range options.Subprotocols {
		if !slices.Contains(upgrader.Subprotocols, subprotocol) {
			upgrader.Subprotocols = append(upgrader.Subprotocols, subprotocol)
		}
	}

	conn, err := upgrader.Upgrade(w, r, options.ResponseHeader)
	if err != nil {
		return nil, err
	}

	return gorillaWebsocketConn{conn: conn}, nil
}

type gorillaWebsocketConn struct {
	conn *websocket.Conn
}

func (c gorillaWebsocketConn) Close() error {
	return c.conn.Close()
}

func (c gorillaWebsocketConn) NextReader() (int, io.Reader, error) {
	mt, r, err := c.conn.NextReader()
	if err != nil && websocket.IsCloseError(
		err,
		WebsocketCloseNormalClosure,
		WebsocketCloseNoStatusReceived,
	) {
		return mt, r, ErrWebsocketClosed
	}

	return mt, r, err
}

func (c gorillaWebsocketConn) WriteJSON(v any) error {
	return c.conn.WriteJSON(v)
}

func (c gorillaWebsocketConn) WriteClose(closeCode int, message string) error {
	return c.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(closeCode, message),
	)
}

func (c gorillaWebsocketConn) Subprotocol() string {
	return c.conn.Subprotocol()
}

func (c gorillaWebsocketConn) SetReadLimit(limit int64) {
	c.conn.SetReadLimit(limit)
}

func (c gorillaWebsocketConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}
