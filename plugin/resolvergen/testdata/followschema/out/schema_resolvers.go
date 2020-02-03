// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package customresolver

import (
	"context"
	"fmt"

	customresolver "github.com/99designs/gqlgen/plugin/resolvergen/testdata/singlefile/out"
)

func (r *queryCustomResolverType) Resolver(ctx context.Context) (*customresolver.Resolver, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *resolverCustomResolverType) Name(ctx context.Context, obj *customresolver.Resolver) (string, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *CustomResolverType) Query() customresolver.QueryResolver { return &queryCustomResolverType{r} }
func (r *CustomResolverType) Resolver() customresolver.ResolverResolver {
	return &resolverCustomResolverType{r}
}

type queryCustomResolverType struct{ *CustomResolverType }
type resolverCustomResolverType struct{ *CustomResolverType }
