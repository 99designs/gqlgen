// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"

	"github.com/99designs/gqlgen/example/federation/products/graph/generated"
	"github.com/99designs/gqlgen/example/federation/products/graph/model"
)

func (r *queryResolver) TopProducts(ctx context.Context, first *int) ([]*model.Product, error) {
	return hats, nil
}

func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
