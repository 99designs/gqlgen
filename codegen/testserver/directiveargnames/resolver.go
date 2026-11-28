package directiveargnames

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
)

type Resolver struct{}

// Root is the resolver for the root field.
func (r *queryResolver) Root(ctx context.Context) (*string, error) {
	panic("not implemented")
}

// Probed is the resolver for the probed field.
func (r *queryResolver) Probed(ctx context.Context, x *string, input *ProbeInput) (*string, error) {
	panic("not implemented")
}

// Literals is the resolver for the literals field.
func (r *queryResolver) Literals(ctx context.Context, x *string) (*string, error) {
	panic("not implemented")
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
