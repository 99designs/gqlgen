//go:generate go run -tags gqlgen_table ../../../../../testdata/gqlgen.go -config gqlgen.yml

package table

import (
	"context"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/models"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver { return Resolver{} }

func (Resolver) Echo(
	ctx context.Context,
	input models.DefaultsInput,
	asMap map[string]any,
) (string, error) {
	return resolvers.EchoDefaults(ctx, input, asMap)
}

// NewSchema returns the executable schema with the resolvers.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers:  Resolver{},
		Directives: DirectiveRoot{Keep: resolvers.Keep, Pass: resolvers.Pass},
	})
}
