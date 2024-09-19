package resolver

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

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

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
/*
	type Resolver struct{}
*/
