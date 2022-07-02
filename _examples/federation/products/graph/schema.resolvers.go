package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/_examples/federation/products/graph/generated"
	"github.com/99designs/gqlgen/_examples/federation/products/graph/model"
)

// TopProducts is the resolver for the topProducts field.
func (r *queryResolver) TopProducts(ctx context.Context, first *int) ([]*model.Product, error) {
	return hats, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
