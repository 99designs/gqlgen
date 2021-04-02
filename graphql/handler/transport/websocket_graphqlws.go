package transport

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

const (
	graphqlwsSubprotocol = "graphql-ws"

	graphqlwsConnectionInitMsg = iota
	graphqlwsConnectionTerminateMsg
	graphqlwsStartMsg
	graphqlwsStopMsg
	graphqlwsConnectionAckMsg
	graphqlwsConnectionErrorMsg
	graphqlwsDataMsg
	graphqlwsErrorMsg
	graphqlwsCompleteMsg
	graphqlwsConnectionKeepAliveMsg
)

type (
	graphqlwsMessageExchanger struct {
		c *websocket.Conn
	}

	graphqlwsMessage struct {
		Payload json.RawMessage      `json:"payload,omitempty"`
		ID      string               `json:"id,omitempty"`
		Type    graphqlwsMessageType `json:"type"`
	}

	graphqlwsMessageType int
)

func (me graphqlwsMessageExchanger) NextMessage() (message, error) {
	_, r, err := me.c.NextReader()
	if err != nil {
		// TODO: should we consider all closure scenarios here for the ws connection?
		// for now we only list the error codes from the original code
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
			err = errWsConnClosed
		}

		return message{}, err
	}

	var graphqlwsMessage graphqlwsMessage
	if err := jsonDecode(r, &graphqlwsMessage); err != nil {
		return message{}, errInvalidMsg
	}

	return graphqlwsMessage.toMessage()
}

func (me graphqlwsMessageExchanger) Send(m *message) error {
	msg := &graphqlwsMessage{}
	if err := msg.fromMessage(m); err != nil {
		return err
	}

	return me.c.WriteJSON(msg)
}

func (t *graphqlwsMessageType) UnmarshalText(text []byte) (err error) {
	switch string(text) {
	default:
		err = fmt.Errorf("invalid message type %s", string(text))
	case "connection_init":
		*t = graphqlwsConnectionInitMsg
	case "connection_terminate":
		*t = graphqlwsConnectionTerminateMsg
	case "start":
		*t = graphqlwsStartMsg
	case "stop":
		*t = graphqlwsStopMsg
	case "connection_ack":
		*t = graphqlwsConnectionAckMsg
	case "connection_error":
		*t = graphqlwsConnectionErrorMsg
	case "data":
		*t = graphqlwsDataMsg
	case "error":
		*t = graphqlwsErrorMsg
	case "complete":
		*t = graphqlwsCompleteMsg
	case "ka":
		*t = graphqlwsConnectionKeepAliveMsg
	}

	return err
}

func (t graphqlwsMessageType) MarshalText() ([]byte, error) {
	var text string
	var err error
	switch t {
	default:
		err = fmt.Errorf("no text representation for message type %d", t)
	case graphqlwsConnectionInitMsg:
		text = "connection_init"
	case graphqlwsConnectionTerminateMsg:
		text = "connection_terminate"
	case graphqlwsStartMsg:
		text = "start"
	case graphqlwsStopMsg:
		text = "stop"
	case graphqlwsConnectionAckMsg:
		text = "connection_ack"
	case graphqlwsConnectionErrorMsg:
		text = "connection_error"
	case graphqlwsDataMsg:
		text = "data"
	case graphqlwsErrorMsg:
		text = "error"
	case graphqlwsCompleteMsg:
		text = "complete"
	case graphqlwsConnectionKeepAliveMsg:
		text = "ka"
	}

	return []byte(text), err
}

func (t graphqlwsMessageType) toMessageType() (mt messageType, err error) {
	switch t {
	default:
		err = fmt.Errorf("unknown message type mapping for %d", t)
	case graphqlwsConnectionInitMsg:
		mt = initMessageType
	case graphqlwsConnectionTerminateMsg:
		mt = connectionCloseMessageType
	case graphqlwsStartMsg:
		mt = startMessageType
	case graphqlwsStopMsg:
		mt = stopMessageType
	case graphqlwsConnectionAckMsg:
		mt = connectionAckMessageType
	case graphqlwsConnectionErrorMsg:
		mt = connectionErrorMessageType
	case graphqlwsDataMsg:
		mt = dataMessageType
	case graphqlwsErrorMsg:
		mt = errorMessageType
	case graphqlwsCompleteMsg:
		mt = completeMessageType
	case graphqlwsConnectionKeepAliveMsg:
		mt = keepAliveMessageType
	}

	return mt, err
}

func (t *graphqlwsMessageType) fromMessageType(mt messageType) (err error) {
	switch mt {
	default:
		err = fmt.Errorf("failed to convert message %s to %s subprotocol", mt, graphqlwsSubprotocol)
	case initMessageType:
		*t = graphqlwsConnectionInitMsg
	case connectionAckMessageType:
		*t = graphqlwsConnectionAckMsg
	case keepAliveMessageType:
		*t = graphqlwsConnectionKeepAliveMsg
	case connectionErrorMessageType:
		*t = graphqlwsConnectionErrorMsg
	case connectionCloseMessageType:
		*t = graphqlwsConnectionTerminateMsg
	case startMessageType:
		*t = graphqlwsStartMsg
	case stopMessageType:
		*t = graphqlwsStopMsg
	case dataMessageType:
		*t = graphqlwsDataMsg
	case completeMessageType:
		*t = graphqlwsCompleteMsg
	case errorMessageType:
		*t = graphqlwsErrorMsg
	}

	return err
}

func (m graphqlwsMessage) toMessage() (message, error) {
	mt, err := m.Type.toMessageType()
	return message{
		payload: m.Payload,
		id:      m.ID,
		t:       mt,
	}, err
}

func (m *graphqlwsMessage) fromMessage(msg *message) (err error) {
	err = m.Type.fromMessageType(msg.t)
	m.ID = msg.id
	m.Payload = msg.payload
	return err
}
