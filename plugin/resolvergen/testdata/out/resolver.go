package customresolver

import (
	"context"
) // THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type CustomResolverType struct{}

func (r *CustomResolverType) Query() QueryResolver {
	return &queryCustomResolverType{r}
}
func (r *CustomResolverType) Resolver() ResolverResolver {
	return &resolverCustomResolverType{r}
}

type queryCustomResolverType struct{ *CustomResolverType }

func (r *queryCustomResolverType) Resolver(ctx context.Context) (*Resolver, error) {
	panic("not implemented")
}

type resolverCustomResolverType struct{ *CustomResolverType }

func (r *resolverCustomResolverType) Name(ctx context.Context, obj *Resolver) (string, error) {
	panic("not implemented")
}
