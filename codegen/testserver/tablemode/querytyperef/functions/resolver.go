//go:generate go run ../../../../../testdata/gqlgen.go -config gqlgen.yml

package functions

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Mutation() MutationResolver { return Resolver{} }

func (Resolver) Ping(context.Context) (*int, error) { return new(1), nil }

func (Resolver) Q(context.Context) (*Query, error) { return &Query{}, nil }

// NewSchema returns the executable schema with the resolvers.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{Resolvers: Resolver{}})
}
