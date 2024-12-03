package generated

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
)

type Resolver struct{}

// EchoIntToInt is the resolver for the echoIntToInt field.
func (r *queryResolver) EchoIntToInt(ctx context.Context, n *int32) (int32, error) {
	panic("not implemented")
}

// EchoInt64ToInt64 is the resolver for the echoInt64ToInt64 field.
func (r *queryResolver) EchoInt64ToInt64(ctx context.Context, n *int) (int, error) {
	panic("not implemented")
}

// EchoIntInputToIntObject is the resolver for the echoIntInputToIntObject field.
func (r *queryResolver) EchoIntInputToIntObject(ctx context.Context, input Input) (*Result, error) {
	panic("not implemented")
}

// EchoInt64InputToInt64Object is the resolver for the echoInt64InputToInt64Object field.
func (r *queryResolver) EchoInt64InputToInt64Object(ctx context.Context, input Input64) (*Result64, error) {
	panic("not implemented")
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

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
