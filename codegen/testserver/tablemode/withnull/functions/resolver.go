//go:generate go run ../../../../../testdata/gqlgen.go -config gqlgen.yml

package functions

import (
	"context"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver { return Resolver{} }

func (Resolver) Posts(ctx context.Context, authorID *string) (*string, error) {
	return new(resolvers.Describe(authorID)), nil
}

// NewSchema returns the executable schema with the resolvers and directives.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers:  Resolver{},
		Directives: DirectiveRoot{FromContext: resolvers.FromContext},
	})
}
