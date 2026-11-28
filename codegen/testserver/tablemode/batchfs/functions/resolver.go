//go:generate go run ../../gen -config gqlgen.yml

package functions

import (
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/batchresolvers"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver { return batchresolvers.FSQuery{} }
func (Resolver) Dog() DogResolver     { return batchresolvers.FSDog{} }
func (Resolver) Cat() CatResolver     { return batchresolvers.FSCat{} }

// NewSchema returns the executable schema with the shared resolvers.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{Resolvers: Resolver{}})
}
