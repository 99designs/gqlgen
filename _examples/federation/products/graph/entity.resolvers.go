package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/_examples/federation/products/graph/generated"
	"github.com/99designs/gqlgen/_examples/federation/products/graph/model"
)

// FindManufacturerByID is the resolver for the findManufacturerByID field.
func (r *entityResolver) FindManufacturerByID(ctx context.Context, id string) (*model.Manufacturer, error) {
	return &model.Manufacturer{
		ID:   id,
		Name: "Millinery " + id,
	}, nil
}

// FindProductByManufacturerIDAndID is the resolver for the findProductByManufacturerIDAndID field.
func (r *entityResolver) FindProductByManufacturerIDAndID(ctx context.Context, manufacturerID string, id string) (*model.Product, error) {
	for _, hat := range hats {
		if hat.ID == id && hat.Manufacturer.ID == manufacturerID {
			return hat, nil
		}
	}
	return nil, nil
}

// FindProductByUpc is the resolver for the findProductByUpc field.
func (r *entityResolver) FindProductByUpc(ctx context.Context, upc string) (*model.Product, error) {
	for _, hat := range hats {
		if hat.Upc == upc {
			return hat, nil
		}
	}
	return nil, nil
}

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
