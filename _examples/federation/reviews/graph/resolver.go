// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
	"context"

	"github.com/99designs/gqlgen/_examples/federation/reviews/graph/model"
)

type Resolver struct{}

func (r *entityResolver) FindProductByManufacturerIDAndID(
	ctx context.Context,
	manufacturerID, id string,
) (*model.Product, error) {
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
