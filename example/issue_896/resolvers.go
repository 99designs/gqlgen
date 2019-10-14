//go:generate go run ../../testdata/gqlgen.go

package issue_896

import (
	"context"
) // THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct{}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Things1(ctx context.Context) ([]*Thing, error) {
	panic("not implemented")
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) Things2(ctx context.Context) (<-chan []*Thing, error) {
	panic("not implemented")
}
