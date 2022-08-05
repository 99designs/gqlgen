package return_values

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"
)

type Resolver struct{}

// // foo
func (r *queryResolver) User(ctx context.Context) (User, error) {
	panic("not implemented")
}

// // foo
func (r *queryResolver) UserPointer(ctx context.Context) (*User, error) {
	panic("not implemented")
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
