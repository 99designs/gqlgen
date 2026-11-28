//go:generate go run -tags gqlgen_table ../../../../../testdata/gqlgen.go -config gqlgen.yml

package table

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Mutation() MutationResolver { return Resolver{} }

func (Resolver) Ping(context.Context) (*int, error) { return new(1), nil }

// NewSchema returns the executable schema with the resolvers.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{Resolvers: Resolver{}})
}
