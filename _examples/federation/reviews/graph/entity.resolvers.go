package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/_examples/federation/reviews/graph/generated"
	"github.com/99designs/gqlgen/_examples/federation/reviews/graph/model"
)

// FindProductByManufacturerIDAndID is the resolver for the findProductByManufacturerIDAndID field.
func (r *entityResolver) FindProductByManufacturerIDAndID(ctx context.Context, manufacturerID string, id string) (*model.Product, error) {
	var productReviews []*model.Review

	for _, review := range reviews {
		if review.Product.ID == id && review.Product.Manufacturer.ID == manufacturerID {
			productReviews = append(productReviews, review)
		}
	}
	return &model.Product{
		ID: id,
		Manufacturer: &model.Manufacturer{
			ID: manufacturerID,
		},
		Reviews: productReviews,
	}, nil
}

// FindUserByID is the resolver for the findUserByID field.
func (r *entityResolver) FindUserByID(ctx context.Context, id string) (*model.User, error) {
	return &model.User{
		ID:   id,
		Host: &model.EmailHost{},
	}, nil
}

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
