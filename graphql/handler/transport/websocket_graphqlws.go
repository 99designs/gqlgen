package transport

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

// https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md
const (
	graphqlwsSubprotocol = "graphql-ws"

	graphqlwsConnectionInitMsg      = graphqlwsMessageType("connection_init")
	graphqlwsConnectionTerminateMsg = graphqlwsMessageType("connection_terminate")
	graphqlwsStartMsg               = graphqlwsMessageType("start")
	graphqlwsStopMsg                = graphqlwsMessageType("stop")
	graphqlwsConnectionAckMsg       = graphqlwsMessageType("connection_ack")
	graphqlwsConnectionErrorMsg     = graphqlwsMessageType("connection_error")
	graphqlwsDataMsg                = graphqlwsMessageType("data")
	graphqlwsErrorMsg               = graphqlwsMessageType("error")
	graphqlwsCompleteMsg            = graphqlwsMessageType("complete")
	graphqlwsConnectionKeepAliveMsg = graphqlwsMessageType("ka")
)

var (
	allGraphqlwsMessageTypes = []graphqlwsMessageType{
		graphqlwsConnectionInitMsg,
		graphqlwsConnectionTerminateMsg,
		graphqlwsStartMsg,
		graphqlwsStopMsg,
		graphqlwsConnectionAckMsg,
		graphqlwsConnectionErrorMsg,
		graphqlwsDataMsg,
		graphqlwsErrorMsg,
		graphqlwsCompleteMsg,
		graphqlwsConnectionKeepAliveMsg,
	}
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

	graphqlwsMessageType string
)

func (me graphqlwsMessageExchanger) NextMessage() (message, error) {
	_, r, err := me.c.NextReader()
	if err != nil {
		return message{}, handleNextReaderError(err)
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
	var found bool
	for _, candidate := range allGraphqlwsMessageTypes {
		if string(candidate) == string(text) {
			*t = candidate
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("invalid message type %s", string(text))
	}

	return err
}

func (t graphqlwsMessageType) MarshalText() ([]byte, error) {
	return []byte(string(t)), nil
}

func (t graphqlwsMessageType) toMessageType() (mt messageType, err error) {
	switch t {
	default:
		err = fmt.Errorf("unknown message type mapping for %s", t)
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
