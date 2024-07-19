package resolver

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"

	nullabledirectives "github.com/99designs/gqlgen/codegen/testserver/nullabledirectives/generated"
)

type Resolver struct{}

// DirectiveSingleNullableArg is the resolver for the directiveSingleNullableArg field.
func (r *queryResolver) DirectiveSingleNullableArg(ctx context.Context, arg1 *string) (*string, error) {
	panic("not implemented")
}

// Query returns nullabledirectives.QueryResolver implementation.
func (r *Resolver) Query() nullabledirectives.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
