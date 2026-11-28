//go:generate go run ../../../../../testdata/gqlgen.go -config gqlgen.yml

package functions

import (
	"context"

	amodel "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliases/a/model"
	bmodel "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliases/b/model"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver { return Resolver{} }

func (Resolver) A(ctx context.Context, in *amodel.Input) (*string, error) {
	return new(resolvers.Describe(in)), nil
}

func (Resolver) B(ctx context.Context, in *bmodel.Input) (*string, error) {
	return new(resolvers.Describe(in)), nil
}

// NewSchema returns the executable schema with the resolvers and directives.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers:  Resolver{},
		Directives: DirectiveRoot{RetypeInput: resolvers.Retype},
	})
}
