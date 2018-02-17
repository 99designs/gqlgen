package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/neelance/errors"
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

type Subscription struct {
	Close func() error
	Next  func(response interface{}) error
}

func errorSubscription(err error) *Subscription {
	return &Subscription{
		Close: func() error { return nil },
		Next: func(response interface{}) error {
			return err
		},
	}
}

func (p *Client) Websocket(query string, options ...Option) *Subscription {
	r := p.mkRequest(query, options...)
	requestBody, err := json.Marshal(r)
	if err != nil {
		return errorSubscription(fmt.Errorf("encode: %s", err.Error()))
	}

	url := strings.Replace(p.url, "http://", "ws://", -1)
	url = strings.Replace(url, "https://", "wss://", -1)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return errorSubscription(fmt.Errorf("dial: %s", err.Error()))
	}

	if err = c.WriteJSON(operationMessage{Type: connectionInitMsg}); err != nil {
		return errorSubscription(fmt.Errorf("init: %s", err.Error()))
	}

	var ack operationMessage
	if err := c.ReadJSON(&ack); err != nil {
		return errorSubscription(fmt.Errorf("ack: %s", err.Error()))
	}
	if ack.Type != connectionAckMsg {
		return errorSubscription(fmt.Errorf("expected ack message, got %#v", ack))
	}

	if err = c.WriteJSON(operationMessage{Type: startMsg, ID: "1", Payload: requestBody}); err != nil {
		return errorSubscription(fmt.Errorf("start: %s", err.Error()))
	}

	return &Subscription{
		Close: c.Close,
		Next: func(response interface{}) error {
			var op operationMessage
			c.ReadJSON(&op)
			if op.Type != dataMsg {
				return fmt.Errorf("expected data message, got %#v", op)
			}

			respDataRaw := map[string]interface{}{}
			err = json.Unmarshal(op.Payload, &respDataRaw)
			if err != nil {
				return fmt.Errorf("decode: %s", err.Error())
			}

			if respDataRaw["errors"] != nil {
				var errs []*errors.QueryError
				if err := unpack(respDataRaw["errors"], errs); err != nil {
					return err
				}
				if len(errs) > 0 {
					return fmt.Errorf("errors: %s", errs)
				}
			}

			return unpack(respDataRaw["data"], response)
		},
	}
}
