package transport

import (
	"io"
	"net/http"
	"time"
)

// WebsocketAcceptOptions contains gqlgen websocket transport options that an
// implementation adapter can apply while accepting a websocket connection.
type WebsocketAcceptOptions struct {
	// ResponseHeader is passed back with the HTTP 101 response.
	ResponseHeader http.Header
	// Subprotocols are the GraphQL websocket subprotocols supported by gqlgen.
	// Implementations should merge these with their own implementation-specific
	// subprotocol configuration when negotiating the websocket connection.
	Subprotocols []string
}

// WebsocketImplementation accepts an HTTP request as a websocket connection.
type WebsocketImplementation interface {
	Accept(
		w http.ResponseWriter,
		r *http.Request,
		options WebsocketAcceptOptions,
	) (WebsocketConn, error)
}

// WebsocketConn is the websocket connection behavior required by the GraphQL
// websocket transport. Adapters for websocket libraries should return
// ErrWebsocketClosed from NextReader when the peer performs a normal close.
type WebsocketConn interface {
	io.Closer
	NextReader() (messageType int, r io.Reader, err error)
	WriteJSON(v any) error
	WriteClose(closeCode int, message string) error
	Subprotocol() string
}

// WebsocketReadLimiter is an optional interface implemented by websocket
// connections that can enforce a maximum read size. If an adapter does not
// implement this interface, the transport does not enforce PayloadReadLimit.
type WebsocketReadLimiter interface {
	SetReadLimit(limit int64)
}

// WebsocketReadDeadliner is an optional interface implemented by websocket
// connections that can enforce read deadlines. If an adapter does not implement
// this interface, PingPongInterval can send pings but cannot enforce missing
// pong timeouts.
type WebsocketReadDeadliner interface {
	SetReadDeadline(t time.Time) error
}

const (
	// WebsocketCloseNormalClosure is the RFC 6455 normal closure status code.
	WebsocketCloseNormalClosure = 1000
	// WebsocketCloseNoStatusReceived is the RFC 6455 no-status-received code.
	WebsocketCloseNoStatusReceived = 1005
	// WebsocketCloseProtocolError is the RFC 6455 protocol error status code.
	WebsocketCloseProtocolError = 1002
)
