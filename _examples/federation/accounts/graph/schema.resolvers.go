package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/_examples/federation/accounts/graph/generated"
	"github.com/99designs/gqlgen/_examples/federation/accounts/graph/model"
)

// Me is the resolver for the me field.
func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	return &model.User{
		ID: "1234",
		Host: &model.EmailHost{
			ID:   "4567",
			Name: "Email Host 4567",
		},
		Email:    "me@example.com",
		Username: "Me",
	}, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
