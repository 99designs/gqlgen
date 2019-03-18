package testserver

import (
	"context"

	introspection1 "github.com/99designs/gqlgen/codegen/testserver/introspection"
	"github.com/99designs/gqlgen/codegen/testserver/invalid-packagename"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct{}

func (r *Resolver) ForcedResolver() ForcedResolverResolver {
	return &forcedResolverResolver{r}
}
func (r *Resolver) ModelMethods() ModelMethodsResolver {
	return &modelMethodsResolver{r}
}
func (r *Resolver) OverlappingFields() OverlappingFieldsResolver {
	return &overlappingFieldsResolver{r}
}
func (r *Resolver) Panics() PanicsResolver {
	return &panicsResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}
func (r *Resolver) User() UserResolver {
	return &userResolver{r}
}

type forcedResolverResolver struct{ *Resolver }

func (r *forcedResolverResolver) Field(ctx context.Context, obj *ForcedResolver) (*Circle, error) {
	panic("not implemented")
}

type modelMethodsResolver struct{ *Resolver }

func (r *modelMethodsResolver) ResolverField(ctx context.Context, obj *ModelMethods) (bool, error) {
	panic("not implemented")
}

type overlappingFieldsResolver struct{ *Resolver }

func (r *overlappingFieldsResolver) OldFoo(ctx context.Context, obj *OverlappingFields) (int, error) {
	panic("not implemented")
}

type panicsResolver struct{ *Resolver }

func (r *panicsResolver) FieldScalarMarshal(ctx context.Context, obj *Panics) ([]MarshalPanic, error) {
	panic("not implemented")
}
func (r *panicsResolver) ArgUnmarshal(ctx context.Context, obj *Panics, u []MarshalPanic) (bool, error) {
	panic("not implemented")
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) InvalidIdentifier(ctx context.Context) (*invalid_packagename.InvalidIdentifier, error) {
	panic("not implemented")
}
func (r *queryResolver) Collision(ctx context.Context) (*introspection1.It, error) {
	panic("not implemented")
}
func (r *queryResolver) MapInput(ctx context.Context, input map[string]interface{}) (*bool, error) {
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
func (r *queryResolver) Shapes(ctx context.Context) ([]Shape, error) {
	panic("not implemented")
}
func (r *queryResolver) ErrorBubble(ctx context.Context) (*Error, error) {
	panic("not implemented")
}
func (r *queryResolver) ModelMethods(ctx context.Context) (*ModelMethods, error) {
	panic("not implemented")
}
func (r *queryResolver) Valid(ctx context.Context) (string, error) {
	panic("not implemented")
}
func (r *queryResolver) User(ctx context.Context, id int) (*User, error) {
	panic("not implemented")
}
func (r *queryResolver) NullableArg(ctx context.Context, arg *int) (*string, error) {
	panic("not implemented")
}
func (r *queryResolver) DirectiveArg(ctx context.Context, arg string) (*string, error) {
	panic("not implemented")
}
func (r *queryResolver) DirectiveNullableArg(ctx context.Context, arg *int, arg2 *int) (*string, error) {
	panic("not implemented")
}
func (r *queryResolver) DirectiveInputNullable(ctx context.Context, arg *InputDirectives) (*string, error) {
	panic("not implemented")
}
func (r *queryResolver) DirectiveInput(ctx context.Context, arg InputDirectives) (*string, error) {
	panic("not implemented")
}
func (r *queryResolver) DirectiveInputType(ctx context.Context, arg InnerInput) (*string, error) {
	panic("not implemented")
}
func (r *queryResolver) InputSlice(ctx context.Context, arg []string) (bool, error) {
	panic("not implemented")
}
func (r *queryResolver) ShapeUnion(ctx context.Context) (ShapeUnion, error) {
	panic("not implemented")
}
func (r *queryResolver) Autobind(ctx context.Context) (*Autobind, error) {
	panic("not implemented")
}
func (r *queryResolver) DeprecatedField(ctx context.Context) (string, error) {
	panic("not implemented")
}
func (r *queryResolver) Overlapping(ctx context.Context) (*OverlappingFields, error) {
	panic("not implemented")
}
func (r *queryResolver) MapStringInterface(ctx context.Context, in map[string]interface{}) (map[string]interface{}, error) {
	panic("not implemented")
}
func (r *queryResolver) Panics(ctx context.Context) (*Panics, error) {
	panic("not implemented")
}
func (r *queryResolver) DefaultScalar(ctx context.Context, arg string) (string, error) {
	panic("not implemented")
}
func (r *queryResolver) Slices(ctx context.Context) (*Slices, error) {
	panic("not implemented")
}
func (r *queryResolver) Fallback(ctx context.Context, arg FallbackToStringEncoding) (FallbackToStringEncoding, error) {
	panic("not implemented")
}
func (r *queryResolver) OptionalUnion(ctx context.Context) (TestUnion, error) {
	panic("not implemented")
}
func (r *queryResolver) ValidType(ctx context.Context) (*ValidType, error) {
	panic("not implemented")
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) Updated(ctx context.Context) (<-chan string, error) {
	panic("not implemented")
}
func (r *subscriptionResolver) InitPayload(ctx context.Context) (<-chan string, error) {
	panic("not implemented")
}

type userResolver struct{ *Resolver }

func (r *userResolver) Friends(ctx context.Context, obj *User) ([]User, error) {
	panic("not implemented")
}
