//go:generate go run ../../gen -config gqlgen.yml

package functions

import (
	"context"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver { return Resolver{} }

func (Resolver) PluginMisplaced(context.Context) (*string, error) { return new("misplaced"), nil }

func (Resolver) PluginObjectOnField(context.Context) (*string, error) { return new("plain"), nil }

func (Resolver) PluginInputObjectOnField(context.Context) (*string, error) {
	return new("plain"), nil
}

func (Resolver) PluginTyped(context.Context) (*string, error) { return new("plain"), nil }

// PluginInput returns the input field, as the input types of the modes are not the same.
func (Resolver) PluginInput(ctx context.Context, in *MarkInput) (*string, error) {
	return new(resolvers.Describe(in.V)), nil
}

// NewSchema returns the executable schema with the resolvers and directives.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers: Resolver{},
		Directives: DirectiveRoot{
			TrimArg:  resolvers.TrimArg,
			OnObject: resolvers.Mark("onObject"),
			OnInput:  resolvers.Mark("onInput"),
			OnField:  resolvers.Mark("onField"),
		},
	})
}
