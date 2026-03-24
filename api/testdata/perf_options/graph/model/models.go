// Package model contains types that are autobound by gqlgen.
// These types should be detected and used instead of being regenerated.
package model

// Metadata is a simple type for testing
type Metadata struct {
	Version   string
	CreatedAt string
}

// Connection represents a network connection.
// gqlgen should detect this type via autobind and NOT regenerate it.
type Connection struct {
	ID       string
	SourceIP string
	DestIP   string
	Port     int
}

// Session contains multiple connections
type Session struct {
	ID          string
	Connections []*Connection
	Active      bool
}

