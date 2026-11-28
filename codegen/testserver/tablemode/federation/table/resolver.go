//go:generate go run -tags gqlgen_table ../../gen -config gqlgen.yml

package table

import (
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/federation/fedresolvers"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver     { return fedresolvers.Query{} }
func (Resolver) Entity() EntityResolver   { return fedresolvers.Entity{} }
func (Resolver) Product() ProductResolver { return fedresolvers.Product{} }

// NewSchema returns the executable schema with the shared resolvers and directives.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{
		Resolvers:  Resolver{},
		Directives: DirectiveRoot{Upper: resolvers.Upper},
	})
}
