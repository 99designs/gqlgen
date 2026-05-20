package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	coderws "github.com/coder/websocket"
)

// CoderWebsocketImplementation adapts github.com/coder/websocket to the gqlgen
// websocket transport.
type CoderWebsocketImplementation struct {
	AcceptOptions coderws.AcceptOptions
}

var _ WebsocketImplementation = CoderWebsocketImplementation{}

func (u CoderWebsocketImplementation) Accept(
	w http.ResponseWriter,
	r *http.Request,
	options WebsocketAcceptOptions,
) (WebsocketConn, error) {
	for key, values := range options.ResponseHeader {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	acceptOptions := u.AcceptOptions
	for _, subprotocol := range options.Subprotocols {
		if !slices.Contains(acceptOptions.Subprotocols, subprotocol) {
			acceptOptions.Subprotocols = append(acceptOptions.Subprotocols, subprotocol)
		}
	}

	conn, err := coderws.Accept(w, r, &acceptOptions)
	if err != nil {
		return nil, err
	}

	return &coderWebsocketConn{conn: conn}, nil
}

type coderWebsocketConn struct {
	conn *coderws.Conn

	mu                sync.Mutex
	readDeadlineTimer *time.Timer
}

var (
	_ WebsocketConn          = (*coderWebsocketConn)(nil)
	_ WebsocketReadLimiter   = (*coderWebsocketConn)(nil)
	_ WebsocketReadDeadliner = (*coderWebsocketConn)(nil)
)

func (c *coderWebsocketConn) Close() error {
	c.clearReadDeadlineTimer()
	return c.conn.CloseNow()
}

func (c *coderWebsocketConn) NextReader() (int, io.Reader, error) {
	messageType, data, err := c.conn.Read(context.Background())
	if err != nil && isCoderNormalClose(err) {
		return int(messageType), nil, ErrWebsocketClosed
	}
	if err != nil {
		return int(messageType), nil, err
	}

	return int(messageType), bytes.NewReader(data), nil
}

func (c *coderWebsocketConn) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return c.conn.Write(context.Background(), coderws.MessageText, data)
}

func (c *coderWebsocketConn) WriteClose(closeCode int, message string) error {
	c.clearReadDeadlineTimer()
	return c.conn.Close(coderws.StatusCode(closeCode), message)
}

func (c *coderWebsocketConn) Subprotocol() string {
	return c.conn.Subprotocol()
}

func (c *coderWebsocketConn) SetReadLimit(limit int64) {
	c.conn.SetReadLimit(limit)
}

func (c *coderWebsocketConn) SetReadDeadline(deadline time.Time) error {
	var closeNow bool

	c.mu.Lock()
	c.stopReadDeadlineTimerLocked()
	if !deadline.IsZero() {
		duration := time.Until(deadline)
		if duration <= 0 {
			closeNow = true
		} else {
			c.readDeadlineTimer = time.AfterFunc(duration, func() {
				_ = c.conn.CloseNow()
			})
		}
	}
	c.mu.Unlock()

	if closeNow {
		return c.conn.CloseNow()
	}
	return nil
}

func (c *coderWebsocketConn) clearReadDeadlineTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopReadDeadlineTimerLocked()
}

func (c *coderWebsocketConn) stopReadDeadlineTimerLocked() {
	if c.readDeadlineTimer != nil {
		c.readDeadlineTimer.Stop()
		c.readDeadlineTimer = nil
	}
}

func isCoderNormalClose(err error) bool {
	switch coderws.CloseStatus(err) {
	case coderws.StatusNormalClosure,
		coderws.StatusNoStatusRcvd:
		return true
	default:
		return false
	}
}
