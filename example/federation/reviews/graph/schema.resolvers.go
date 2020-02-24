package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/example/federation/reviews/graph/generated"
	"github.com/99designs/gqlgen/example/federation/reviews/graph/model"
)

func (r *productResolver) Reviews(ctx context.Context, obj *model.Product) ([]*model.Review, error) {
	var res []*model.Review

	for _, review := range reviews {
		if review.Product.Upc == obj.Upc {
			res = append(res, review)
		}
	}

	return res, nil
}

func (r *userResolver) Reviews(ctx context.Context, obj *model.User) ([]*model.Review, error) {
	var res []*model.Review

	for _, review := range reviews {
		if review.Author.ID == obj.ID {
			res = append(res, review)
		}
	}

	return res, nil
}

// Product returns generated.ProductResolver implementation.
func (r *Resolver) Product() generated.ProductResolver { return &productResolver{r} }

// User returns generated.UserResolver implementation.
func (r *Resolver) User() generated.UserResolver { return &userResolver{r} }

type productResolver struct{ *Resolver }
type userResolver struct{ *Resolver }
