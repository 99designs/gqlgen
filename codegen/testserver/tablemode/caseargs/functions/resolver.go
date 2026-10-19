//go:generate go run ../../../../../testdata/gqlgen.go -config gqlgen.yml

package functions

import (
	"context"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/models"
	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

func (Resolver) Query() QueryResolver { return Resolver{} }

func (Resolver) CaseArgs(context.Context) (*models.CaseArgs, error) {
	return &models.CaseArgs{}, nil
}

// NewSchema returns the executable schema with the resolvers.
func NewSchema() graphql.ExecutableSchema {
	return NewExecutableSchema(Config{Resolvers: Resolver{}})
}
