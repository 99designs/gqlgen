// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"

	"github.com/99designs/gqlgen/example/federation/products/graph/generated"
	"github.com/99designs/gqlgen/example/federation/products/graph/model"
)

func (r *entityResolver) FindProductByUpc(ctx context.Context, upc string) (*model.Product, error) {
	for _, h := range hats {
		if h.Upc == upc {
			return h, nil
		}
	}
	return nil, nil
}

func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
