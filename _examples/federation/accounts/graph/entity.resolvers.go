package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/_examples/federation/accounts/graph/generated"
	"github.com/99designs/gqlgen/_examples/federation/accounts/graph/model"
)

func (r *entityResolver) FindEmailHostByID(ctx context.Context, id string) (*model.EmailHost, error) {
	return &model.EmailHost{
		ID:   id,
		Name: "Email Host " + id,
	}, nil
}

func (r *entityResolver) FindUserByID(ctx context.Context, id string) (*model.User, error) {
	name := "User " + id
	if id == "1234" {
		name = "Me"
	}

	return &model.User{
		ID:       id,
		Username: name,
	}, nil
}

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
