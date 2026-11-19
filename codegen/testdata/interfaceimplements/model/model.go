package model

// Node is bound to an empty interface, so every Go type satisfies it,
// including a pointer to an interface.
type Node interface{}

// Captioned is a GraphQL interface that implements Node, bound to a Go
// interface.
type Captioned interface {
	GetID() string
	GetCaption() string
}

type Post struct {
	ID      string
	Caption string
}

func (p Post) GetID() string      { return p.ID }
func (p Post) GetCaption() string { return p.Caption }
