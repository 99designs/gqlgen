package generated

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
)

type Resolver struct{}

// OverflowInt32ButReturnInt64 is the resolver for the overflowInt32ButReturnInt64 field.
func (r *queryResolver) OverflowInt32ButReturnInt64(ctx context.Context, sign Sign) (*int, error) {
	panic("not implemented")
}

// OverflowInt32 is the resolver for the overflowInt32 field.
func (r *queryResolver) OverflowInt32(ctx context.Context, sign Sign) (*int32, error) {
	panic("not implemented")
}

// EchoInt32In is the resolver for the echoInt32In field.
func (r *queryResolver) EchoInt32In(ctx context.Context, n *int32) (int32, error) {
	panic("not implemented")
}

// EchoInt64In is the resolver for the echoInt64In field.
func (r *queryResolver) EchoInt64In(ctx context.Context, n *int) (int32, error) {
	panic("not implemented")
}

// EchoInt32 is the resolver for the echoInt32 field.
func (r *queryResolver) EchoInt32(ctx context.Context, input Input) (*Result, error) {
	panic("not implemented")
}

// EchoInt64 is the resolver for the echoInt64 field.
func (r *queryResolver) EchoInt64(ctx context.Context, input Input64) (*Result64, error) {
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
