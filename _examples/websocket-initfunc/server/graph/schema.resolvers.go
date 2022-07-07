package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/gqlgen/_examples/websocket-initfunc/server/graph/generated"
)

// PostMessageTo is the resolver for the postMessageTo field.
func (r *mutationResolver) PostMessageTo(ctx context.Context, subscriber string, content string) (string, error) {
	panic(fmt.Errorf("not implemented"))
}

// Subscribe is the resolver for the subscribe field.
func (r *subscriptionResolver) Subscribe(ctx context.Context, subscriber string) (<-chan string, error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
