//go:generate go run ../gen -config gqlgen.yml

package functions

import (
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver                 { return resolvers.Query{} }
func (Resolver) Mutation() MutationResolver           { return resolvers.Mutation{} }
func (Resolver) Human() HumanResolver                 { return resolvers.Human{} }
func (Resolver) Droid() DroidResolver                 { return resolvers.Droid{} }
func (Resolver) Starship() StarshipResolver           { return resolvers.Starship{} }
func (Resolver) ResolvedInput() ResolvedInputResolver { return resolvers.ResolvedInput{} }
func (Resolver) Subscription() SubscriptionResolver   { return resolvers.Subscription{} }

// NewSchema returns the executable schema with the shared resolvers and directives.
func NewSchema() graphql.ExecutableSchema { return NewSchemaFor(nil) }

// NewSchemaFor is NewSchema with schema in the Config, which the executor validates
// requests against and answers introspection from in place of the generated schema. A
// nil schema keeps the generated one.
func NewSchemaFor(schema *ast.Schema) graphql.ExecutableSchema {
	cfg := Config{
		Resolvers: Resolver{},
		Directives: DirectiveRoot{
			Upper:           resolvers.Upper,
			Auth:            resolvers.Auth,
			Length:          resolvers.Length,
			Nullify:         resolvers.Nullify,
			ValidReview:     resolvers.ValidReview,
			Trace:           resolvers.Trace,
			LogQuery:        resolvers.LogQuery,
			LogMutation:     resolvers.LogMutation,
			Tagged:          resolvers.Tagged,
			ArgNote:         resolvers.ArgNote,
			Explode:         resolvers.Explode,
			LogSubscription: resolvers.LogSubscription,
			TrimArg:         resolvers.TrimArg,
			Note:            resolvers.Note,
			Nulled:          resolvers.Nulled,
			Probe:           resolvers.Probe,
			Order:           resolvers.Order,
			Retype:          resolvers.Retype,
			// Missing is left out, as a directive without an implementation.
		},
		Schema: schema,
	}
	cfg.Complexity.Human.Height = resolvers.HeightComplexity
	return NewExecutableSchema(cfg)
}
