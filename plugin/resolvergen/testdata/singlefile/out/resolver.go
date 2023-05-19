package customresolver

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"
)

type CustomResolverType struct{}

// Resolver is the resolver for the resolver field.
func (r *queryCustomResolverType) Resolver(ctx context.Context) (*Resolver, error) {
	panic("not implemented")
}

// Name is the resolver for the name field.
func (r *resolverCustomResolverType) Name(ctx context.Context, obj *Resolver) (string, error) {
	panic("not implemented")
}

// Query returns QueryResolver implementation.
func (r *CustomResolverType) Query() QueryResolver { return &queryCustomResolverType{r} }

// Resolver returns ResolverResolver implementation.
func (r *CustomResolverType) Resolver() ResolverResolver { return &resolverCustomResolverType{r} }

type queryCustomResolverType struct{ *CustomResolverType }
type resolverCustomResolverType struct{ *CustomResolverType }
