package chat

import (
	"time"
)

type Chatroom struct {
	Name      string
	Messages  []Message
	Observers map[string]chan Message
}

func (c *Chatroom) ID() string { return "C" + c.Name }

type Message struct {
	ID        string
	Text      string
	CreatedBy string
	CreatedAt time.Time
}
