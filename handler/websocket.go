package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/validation"
)

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

type wsConnection struct {
	conn           *websocket.Conn
	exec           graphql.ExecutableSchema
	active         map[string]context.CancelFunc
	keepAliveTimer *time.Timer
	mu             sync.Mutex
	cfg            *Config
}

func connectWs(exec graphql.ExecutableSchema, w http.ResponseWriter, r *http.Request, cfg *Config) {
	ws, err := cfg.upgrader.Upgrade(w, r, http.Header{
		"Sec-Websocket-Protocol": []string{"graphql-ws"},
	})
	if err != nil {
		log.Printf("unable to upgrade %T to websocket %s: ", w, err.Error())
		sendErrorf(w, http.StatusBadRequest, "unable to upgrade")
		return
	}

	conn := wsConnection{
		active: map[string]context.CancelFunc{},
		exec:   exec,
		conn:   ws,
		cfg:    cfg,
	}

	initMessage, ok := conn.init()
	if !ok {
		return
	}

	// When the websocket connection is initialized, and a onConnect
	// function is defined, then we should pass the payload data up to
	// the handler.
	connectionParams := map[string]interface{}{}
	if err := json.Unmarshal(initMessage.Payload, &connectionParams); err != nil {
		conn.sendConnectionError("invalid json")
		return
	}

	next := func(ctx context.Context, params map[string]interface{}) error {
		return conn.run(ctx)
	}

	if err := cfg.onConnectHook(r.Context(), connectionParams, next); err != nil {
		// TODO: handle the error somehow?
		return
	}
}

// init retrieves the connection init message that signifies a valid established
// subscription connection.
func (c *wsConnection) init() (*operationMessage, bool) {
	message := c.readOp()
	if message == nil {
		c.close(websocket.CloseProtocolError, "decoding error")
		return nil, false
	}

	switch message.Type {
	case connectionInitMsg:
		c.write(&operationMessage{Type: connectionAckMsg})
	case connectionTerminateMsg:
		c.close(websocket.CloseNormalClosure, "terminated")
		return nil, false
	default:
		c.sendConnectionError("unexpected message %s", message.Type)
		c.close(websocket.CloseProtocolError, "unexpected message")
		return nil, false
	}

	return message, true
}

func (c *wsConnection) write(msg *operationMessage) {
	c.mu.Lock()
	c.conn.WriteJSON(msg) // TODO: handle error

	// Reset the keep alive timer if it's been setup.
	if c.cfg.connectionKeepAliveTimeout != 0 && c.keepAliveTimer != nil {
		c.keepAliveTimer.Reset(c.cfg.connectionKeepAliveTimeout)
	}

	c.mu.Unlock()
}

func (c *wsConnection) keepAlive(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if !c.keepAliveTimer.Stop() {
				<-c.keepAliveTimer.C
			}
			return
		case <-c.keepAliveTimer.C:
			// We don't reset the timer here, because the `c.write` command
			// will reset the timer anyways.
			c.write(&operationMessage{Type: connectionKeepAliveMsg})
		}
	}
}

func (c *wsConnection) run(ctx context.Context) error {
	// We create a cancellation that will shutdown the keep-alive when we leave
	// this function.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a timer that will fire every interval if a write hasn't been made
	// to keep the connection alive.
	if c.cfg.connectionKeepAliveTimeout != 0 {
		// Create the new timer that will fire `c.cfg.connectionKeepAliveTimeout`
		// from now.
		c.mu.Lock()
		c.keepAliveTimer = time.NewTimer(c.cfg.connectionKeepAliveTimeout)
		c.mu.Unlock()

		// Launch the keepAlive manager. This will exit when the context is
		// canceled.
		go c.keepAlive(ctx)
	}

	for {
		message := c.readOp()
		if message == nil {
			return errors.Errorf("message was nil")
		}

		switch message.Type {
		case startMsg:
			if !c.subscribe(ctx, message) {
				return errors.Errorf("subscription failed")
			}
		case stopMsg:
			c.mu.Lock()
			closer := c.active[message.ID]
			c.mu.Unlock()
			if closer == nil {
				c.sendError(message.ID, errors.Errorf("%s is not running, cannot stop", message.ID))
				continue
			}

			closer()
		case connectionTerminateMsg:
			c.close(websocket.CloseNormalClosure, "terminated")
			return nil
		default:
			c.sendConnectionError("unexpected message %s", message.Type)
			c.close(websocket.CloseProtocolError, "unexpected message")
			return errors.Errorf("unexpected message: %s", message.Type)
		}
	}
}

func (c *wsConnection) subscribe(ctx context.Context, message *operationMessage) bool {
	var reqParams params
	if err := json.Unmarshal(message.Payload, &reqParams); err != nil {
		c.sendConnectionError("invalid json")
		return false
	}

	doc, qErr := query.Parse(reqParams.Query)
	if qErr != nil {
		c.sendError(message.ID, qErr)
		return true
	}

	errs := validation.Validate(c.exec.Schema(), doc)
	if len(errs) != 0 {
		c.sendError(message.ID, errs...)
		return true
	}

	op, err := doc.GetOperation(reqParams.OperationName)
	if err != nil {
		c.sendError(message.ID, errors.Errorf("%s", err.Error()))
		return true
	}

	reqCtx := c.cfg.newRequestContext(doc, reqParams.Query, reqParams.Variables)
	ctx := graphql.WithRequestContext(c.ctx, reqCtx)

	if op.Type != query.Subscription {
		var result *graphql.Response
		if op.Type == query.Query {
			result = c.exec.Query(ctx, op)
		} else {
			result = c.exec.Mutation(ctx, op)
		}

		c.sendData(message.ID, result)
		c.write(&operationMessage{ID: message.ID, Type: completeMsg})
		return true
	}

	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.active[message.ID] = cancel
	c.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				userErr := reqCtx.Recover(ctx, r)
				c.sendError(message.ID, &errors.QueryError{Message: userErr.Error()})
			}
		}()

		// TODO: create new context per subscription operation

		next := c.exec.Subscription(ctx, op)
		for result := next(); result != nil; result = next() {
			c.sendData(message.ID, result)
		}

		c.write(&operationMessage{ID: message.ID, Type: completeMsg})

		c.mu.Lock()
		delete(c.active, message.ID)
		c.mu.Unlock()
		cancel()
	}()

	return true
}

func (c *wsConnection) sendData(id string, response *graphql.Response) {
	b, err := json.Marshal(response)
	if err != nil {
		c.sendError(id, errors.Errorf("unable to encode json response: %s", err.Error()))
		return
	}

	c.write(&operationMessage{Type: dataMsg, ID: id, Payload: b})
}

func (c *wsConnection) sendError(id string, errors ...*errors.QueryError) {
	var errs []error
	for _, err := range errors {
		errs = append(errs, err)
	}
	b, err := json.Marshal(errs)
	if err != nil {
		panic(err)
	}
	c.write(&operationMessage{Type: errorMsg, ID: id, Payload: b})
}

func (c *wsConnection) sendConnectionError(format string, args ...interface{}) {
	b, err := json.Marshal(&graphql.ResolverError{Message: fmt.Sprintf(format, args...)})
	if err != nil {
		panic(err)
	}

	c.write(&operationMessage{Type: connectionErrorMsg, Payload: b})
}

func (c *wsConnection) readOp() *operationMessage {
	message := operationMessage{}
	if err := c.conn.ReadJSON(&message); err != nil {
		c.sendConnectionError("invalid json")
		return nil
	}
	return &message
}

func (c *wsConnection) close(closeCode int, message string) {
	c.mu.Lock()
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, message))
	c.mu.Unlock()
	_ = c.conn.Close()
}
