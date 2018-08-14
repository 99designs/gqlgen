//go:generate gorunpkg github.com/99designs/gqlgen

package testserver

import (
	context "context"

	introspection1 "github.com/99designs/gqlgen/codegen/testserver/introspection"
	invalid_packagename "github.com/99designs/gqlgen/codegen/testserver/invalid-packagename"
)

type Resolver struct{}

func (r *Resolver) ForcedResolver() ForcedResolverResolver {
	return &forcedResolverResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type forcedResolverResolver struct{ *Resolver }

func (r *forcedResolverResolver) Field(ctx context.Context, obj *ForcedResolver) (*Circle, error) {
	panic("not implemented")
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) InvalidIdentifier(ctx context.Context) (*invalid_packagename.InvalidIdentifier, error) {
	panic("not implemented")
}
func (r *queryResolver) Collision(ctx context.Context) (*introspection1.It, error) {
	panic("not implemented")
}
func (r *queryResolver) MapInput(ctx context.Context, input *map[string]interface{}) (*bool, error) {
	panic("not implemented")
}
func (r *queryResolver) Recursive(ctx context.Context, input *RecursiveInputSlice) (*bool, error) {
	panic("not implemented")
}
func (r *queryResolver) NestedInputs(ctx context.Context, input [][]*OuterInput) (*bool, error) {
	panic("not implemented")
}
func (r *queryResolver) NestedOutputs(ctx context.Context) ([][]*OuterObject, error) {
	panic("not implemented")
}
func (r *queryResolver) Keywords(ctx context.Context, input *Keywords) (bool, error) {
	panic("not implemented")
}
func (r *queryResolver) Shapes(ctx context.Context) ([]*Shape, error) {
	panic("not implemented")
}
func (r *queryResolver) KeywordArgs(ctx context.Context, breakArg string, defaultArg string, funcArg string, interfaceArg string, selectArg string, caseArg string, deferArg string, goArg string, mapArg string, structArg string, chanArg string, elseArg string, gotoArg string, packageArg string, switchArg string, constArg string, fallthroughArg string, ifArg string, rangeArg string, typeArg string, continueArg string, forArg string, importArg string, returnArg string, varArg string) (bool, error) {
	panic("not implemented")
}
