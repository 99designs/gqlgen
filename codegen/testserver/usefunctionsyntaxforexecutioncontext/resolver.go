package usefunctionsyntaxforexecutioncontext

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
)

type Resolver struct{}

// CreateUser is the resolver for the createUser field.
func (r *mutationResolver) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
	panic("not implemented")
}

// DeleteUser is the resolver for the deleteUser field.
func (r *mutationResolver) DeleteUser(ctx context.Context, id string) (*MutationResponse, error) {
	panic("not implemented")
}

// GetUser is the resolver for the getUser field.
func (r *queryResolver) GetUser(ctx context.Context, id string) (*User, error) {
	panic("not implemented")
}

// ListUsers is the resolver for the listUsers field.
func (r *queryResolver) ListUsers(ctx context.Context, filter *UserFilter) ([]*User, error) {
	panic("not implemented")
}

// GetEntity is the resolver for the getEntity field.
func (r *queryResolver) GetEntity(ctx context.Context, id string) (Entity, error) {
	panic("not implemented")
}

// UserCreated is the resolver for the userCreated field.
func (r *subscriptionResolver) UserCreated(ctx context.Context) (<-chan *User, error) {
	panic("not implemented")
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Subscription returns SubscriptionResolver implementation.
func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
