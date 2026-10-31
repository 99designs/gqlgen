package disableconcurrency

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
)

type Resolver struct{}

// Methods is the resolver for the methods field.
func (r *queryResolver) Methods(ctx context.Context) (*Methods, error) {
	panic("not implemented")
}

// InlineObject is the resolver for the inlineObject field.
func (r *queryResolver) InlineObject(ctx context.Context) (*InlineObject, error) {
	panic("not implemented")
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
