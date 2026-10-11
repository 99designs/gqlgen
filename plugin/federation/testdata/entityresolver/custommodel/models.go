package custommodel

import (
	"errors"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

// OutputID binds an output ID field without changing the default string input type.
type OutputID string

// MarshalGQL implements graphql.Marshaler.
func (id OutputID) MarshalGQL(w io.Writer) {
	graphql.MarshalID(string(id)).MarshalGQL(w)
}

// UnmarshalGQL rejects using the output scalar to decode federation input arguments.
func (*OutputID) UnmarshalGQL(any) error {
	return errors.New("output ID must not decode an entity representation")
}

// Owner contains a nested key bound to an output scalar.
type Owner struct {
	ID OutputID
}

// Entity binds scalar, nested and nullable key fields to output scalars.
type Entity struct {
	ID         OutputID
	Owner      *Owner
	OptionalID *OutputID
}

// IsEntity implements the federation entity interface.
func (Entity) IsEntity() {}

// MultiEntity has the same key fields with a batch entity resolver.
type MultiEntity Entity

// IsEntity implements the federation entity interface.
func (MultiEntity) IsEntity() {}
