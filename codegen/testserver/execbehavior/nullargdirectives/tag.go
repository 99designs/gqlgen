package nullargdirectives

import (
	"github.com/99designs/gqlgen/graphql"
)

// Tag is the Go type of the Tag scalar: an interface, so that a directive that takes a
// Tag receives nil when the argument is absent.
type Tag interface {
	Tag() string
}

type tag string

func (t tag) Tag() string { return string(t) }

func MarshalTag(t Tag) graphql.Marshaler {
	return graphql.MarshalString(t.Tag())
}

func UnmarshalTag(v any) (Tag, error) {
	s, err := graphql.UnmarshalString(v)
	if err != nil {
		return nil, err
	}
	return tag(s), nil
}
