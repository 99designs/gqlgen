package transport

import (
	"encoding/json"
	"errors"
)

const (
	initMessageType messageType = iota
	connectionAckMessageType
	keepAliveMessageType
	connectionErrorMessageType
	connectionCloseMessageType
	startMessageType
	stopMessageType
	dataMessageType
	completeMessageType
	errorMessageType
	pingMessageType
	pongMessageType
)

var (
	supportedSubprotocols = []string{
		graphqlwsSubprotocol,
		graphqltransportwsSubprotocol,
	}

	errInvalidMsg = errors.New("invalid message received")
)

type (
	messageType int
	message     struct {
		payload json.RawMessage
		id      string
		t       messageType
	}
	messageExchanger interface {
		NextMessage() (message, error)
		Send(m *message) error
	}
)

func (t messageType) String() string {
	var text string
	switch t {
	default:
		text = "unknown"
	case initMessageType:
		text = "init"
	case connectionAckMessageType:
		text = "connection ack"
	case keepAliveMessageType:
		text = "keep alive"
	case connectionErrorMessageType:
		text = "connection error"
	case connectionCloseMessageType:
		text = "connection close"
	case startMessageType:
		text = "start"
	case stopMessageType:
		text = "stop subscription"
	case dataMessageType:
		text = "data"
	case completeMessageType:
		text = "complete"
	case errorMessageType:
		text = "error"
	case pingMessageType:
		text = "ping"
	case pongMessageType:
		text = "pong"
	}
	return text
}
