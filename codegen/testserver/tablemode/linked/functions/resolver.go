//go:generate go run ../../gen -config gqlgen.yml

package functions

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/linked/model"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver       { return Resolver{} }
func (Resolver) Mutation() MutationResolver { return Resolver{} }

func (Resolver) PluginRequired(context.Context) (string, error) { return "required", nil }

func (Resolver) Plan(ctx context.Context, id string) (*model.Subscription, error) {
	if id == "x" {
		return nil, errors.New("no plan")
	}
	return &model.Subscription{Name: new("basic")}, nil
}

func (Resolver) Name(context.Context) (*string, error) { return new("query name"), nil }

func (Resolver) Ping(context.Context) (*int, error) { return new(1), nil }

// NewSchema returns the executable schema with the resolvers and directives.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers:  Resolver{},
		Directives: DirectiveRoot{Upper: resolvers.Upper},
	})
}
