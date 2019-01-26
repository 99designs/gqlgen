package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/gqlerror"
)

const (
	connectionInitMsg = "connection_init" // Client -> Server
	startMsg          = "start"           // Client -> Server
	connectionAckMsg  = "connection_ack"  // Server -> Client
	dataMsg           = "data"            // Server -> Client
	errorMsg          = "error"           // Server -> Client
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
	return p.WebsocketWithPayload(query, nil, options...)
}

func (p *Client) WebsocketWithPayload(query string, initPayload map[string]interface{}, options ...Option) *Subscription {
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

	initMessage := operationMessage{Type: connectionInitMsg}
	if initPayload != nil {
		initMessage.Payload, err = json.Marshal(initPayload)
		if err != nil {
			return errorSubscription(fmt.Errorf("parse payload: %s", err.Error()))
		}
	}

	if err = c.WriteJSON(initMessage); err != nil {
		return errorSubscription(fmt.Errorf("init: %s", err.Error()))
	}

	var ack operationMessage
	if err = c.ReadJSON(&ack); err != nil {
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
			err := c.ReadJSON(&op)
			if err != nil {
				return err
			}
			if op.Type != dataMsg {
				if op.Type == errorMsg {
					return fmt.Errorf(string(op.Payload))
				} else {
					return fmt.Errorf("expected data message, got %#v", op)
				}
			}

			respDataRaw := map[string]interface{}{}
			err = json.Unmarshal(op.Payload, &respDataRaw)
			if err != nil {
				return fmt.Errorf("decode: %s", err.Error())
			}

			if respDataRaw["errors"] != nil {
				var errs []*gqlerror.Error
				if err = unpack(respDataRaw["errors"], &errs); err != nil {
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
