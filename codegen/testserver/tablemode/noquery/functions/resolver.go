//go:generate go run ../../../../../testdata/gqlgen.go -config gqlgen.yml

package functions

import (
	"context"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Mutation() MutationResolver { return Resolver{} }

func (Resolver) Ping(context.Context) (*int, error) { return new(1), nil }

// NewSchema returns the executable schema with the resolvers.
func NewSchema() graphql.ExecutableSchema { return NewSchemaFor(nil) }

// NewSchemaFor is NewSchema with schema in the Config, which the executor validates
// requests against and answers introspection from.
func NewSchemaFor(schema *ast.Schema) graphql.ExecutableSchema {
	return NewExecutableSchema(Config{Resolvers: Resolver{}, Schema: schema})
}
