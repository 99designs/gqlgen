package subscriptionwithcontext

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type Resolver struct{}

// Noop is the resolver for the noop field.
func (r *queryResolver) Noop(ctx context.Context) (*bool, error) {
	panic("not implemented")
}

// Marked is the resolver for the marked field.
func (r *subscriptionResolver) Marked(ctx context.Context) (<-chan graphql.Event[string], error) {
	panic("not implemented")
}

// Unmarked is the resolver for the unmarked field.
func (r *subscriptionResolver) Unmarked(ctx context.Context) (<-chan string, error) {
	panic("not implemented")
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Subscription returns SubscriptionResolver implementation.
func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
