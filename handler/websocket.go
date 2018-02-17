package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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
	connectionKeepAliveMsg = "ka"                   // Server -> Client
	dataMsg                = "data"                 // Server -> Client
	errorMsg               = "error"                // Server -> Client
	completeMsg            = "complete"             // Server -> Client
)

type operationMessage struct {
	Payload json.RawMessage `json:"payload,omitempty"`
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
}

type wsConnection struct {
	ctx    context.Context
	conn   *websocket.Conn
	exec   graphql.ExecutableSchema
	active map[string]context.CancelFunc
	mu     sync.Mutex
}

func connectWs(exec graphql.ExecutableSchema, w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader) {
	ws, err := upgrader.Upgrade(w, r, http.Header{
		"Sec-Websocket-Protocol": []string{"graphql-ws"},
	})
	if err != nil {
		log.Printf("unable to upgrade connection to websocket %s: ", err.Error())
		sendErrorf(w, http.StatusBadRequest, "unable to upgrade")
		return
	}

	conn := wsConnection{
		active: map[string]context.CancelFunc{},
		exec:   exec,
		conn:   ws,
		ctx:    r.Context(),
	}

	if !conn.init() {
		return
	}

	conn.run()
}

func (c *wsConnection) init() bool {
	message := c.readOp()
	if message == nil {
		c.close(websocket.CloseProtocolError, "decoding error")
		return false
	}

	switch message.Type {
	case connectionInitMsg:
		c.write(&operationMessage{Type: connectionAckMsg})
	case connectionTerminateMsg:
		c.close(websocket.CloseNormalClosure, "terminated")
		return false
	default:
		c.sendConnectionError("unexpected message %s", message.Type)
		c.close(websocket.CloseProtocolError, "unexpected message")
		return false
	}

	return true
}

func (c *wsConnection) write(msg *operationMessage) {
	c.mu.Lock()
	c.conn.WriteJSON(msg)
	c.mu.Unlock()
}

func (c *wsConnection) run() {
	for {
		message := c.readOp()
		if message == nil {
			return
		}

		switch message.Type {
		case startMsg:
			if !c.subscribe(message) {
				return
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
			return
		default:
			c.sendConnectionError("unexpected message %s", message.Type)
			c.close(websocket.CloseProtocolError, "unexpected message")
			return
		}
	}
}

func (c *wsConnection) subscribe(message *operationMessage) bool {
	var params params
	if err := json.Unmarshal(message.Payload, &params); err != nil {
		c.sendConnectionError("invalid json")
		return false
	}

	doc, qErr := query.Parse(params.Query)
	if qErr != nil {
		c.sendError(message.ID, qErr)
		return true
	}

	errs := validation.Validate(c.exec.Schema(), doc)
	if len(errs) != 0 {
		c.sendError(message.ID, errs...)
		return true
	}

	op, err := doc.GetOperation(params.OperationName)
	if err != nil {
		c.sendError(message.ID, errors.Errorf("%s", err.Error()))
		return true
	}

	if op.Type != query.Subscription {
		var result *graphql.Response
		if op.Type == query.Query {
			result = c.exec.Query(c.ctx, doc, params.Variables, op)
		} else {
			result = c.exec.Mutation(c.ctx, doc, params.Variables, op)
		}

		c.sendData(message.ID, result)
		c.write(&operationMessage{ID: message.ID, Type: completeMsg})
		return true
	}

	ctx, cancel := context.WithCancel(c.ctx)
	c.mu.Lock()
	c.active[message.ID] = cancel
	c.mu.Unlock()
	go func() {
		for result := range c.exec.Subscription(ctx, doc, params.Variables, op) {
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
	var b bytes.Buffer
	response.MarshalGQL(&b)

	c.write(&operationMessage{Type: dataMsg, ID: id, Payload: b.Bytes()})
}

func (c *wsConnection) sendError(id string, errors ...*errors.QueryError) {
	writer := graphql.MarshalErrors(errors)
	var b bytes.Buffer
	writer.MarshalGQL(&b)

	c.write(&operationMessage{Type: errorMsg, ID: id, Payload: b.Bytes()})
}

func (c *wsConnection) sendConnectionError(format string, args ...interface{}) {
	writer := graphql.MarshalError(&errors.QueryError{Message: fmt.Sprintf(format, args...)})
	var b bytes.Buffer
	writer.MarshalGQL(&b)

	c.write(&operationMessage{Type: connectionErrorMsg, Payload: b.Bytes()})
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
