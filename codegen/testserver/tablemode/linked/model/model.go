// Package model holds the Go types of the linked schema.
package model

import "errors"

// Subscription is an ordinary object type, which the schema does not declare as a root.
type Subscription struct {
	Name *string
}

// Fail fails, so that the response shows the path of its error.
func (s *Subscription) Fail() (*string, error) { return nil, errors.New("plan fails") }
