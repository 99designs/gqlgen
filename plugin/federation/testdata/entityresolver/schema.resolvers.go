package entityresolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver/generated"
)

func (r *queryResolver) Hello(ctx context.Context) (*generated.Hello, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) World(ctx context.Context) (*generated.World, error) {
	panic(fmt.Errorf("not implemented"))
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
