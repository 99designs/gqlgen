package followschema

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"

	introspection1 "github.com/99designs/gqlgen/codegen/testserver/followschema/introspection"
	invalid_packagename "github.com/99designs/gqlgen/codegen/testserver/followschema/invalid-packagename"
	"github.com/99designs/gqlgen/codegen/testserver/followschema/otherpkg"
)

type Resolver struct{}

// ID is the resolver for the id field.
func (r *backedByInterfaceResolver) ID(ctx context.Context, obj BackedByInterface) (string, error) {
	panic("not implemented")
}

// Values is the resolver for the values field.
func (r *deferModelResolver) Values(ctx context.Context, obj *DeferModel) ([]string, error) {
	panic("not implemented")
}

// A is the resolver for the a field.
func (r *errorsResolver) A(ctx context.Context, obj *Errors) (*Error, error) {
	panic("not implemented")
}

// B is the resolver for the b field.
func (r *errorsResolver) B(ctx context.Context, obj *Errors) (*Error, error) {
	panic("not implemented")
}

// C is the resolver for the c field.
func (r *errorsResolver) C(ctx context.Context, obj *Errors) (*Error, error) {
	panic("not implemented")
}

// D is the resolver for the d field.
func (r *errorsResolver) D(ctx context.Context, obj *Errors) (*Error, error) {
	panic("not implemented")
}

// E is the resolver for the e field.
func (r *errorsResolver) E(ctx context.Context, obj *Errors) (*Error, error) {
	panic("not implemented")
}

// Field is the resolver for the field field.
func (r *forcedResolverResolver) Field(ctx context.Context, obj *ForcedResolver) (*Circle, error) {
	panic("not implemented")
}

// ResolverField is the resolver for the resolverField field.
func (r *modelMethodsResolver) ResolverField(ctx context.Context, obj *ModelMethods) (bool, error) {
	panic("not implemented")
}

// DefaultInput is the resolver for the defaultInput field.
func (r *mutationResolver) DefaultInput(ctx context.Context, input DefaultInput) (*DefaultParametersMirror, error) {
	panic("not implemented")
}

// OverrideValueViaInput is the resolver for the overrideValueViaInput field.
func (r *mutationResolver) OverrideValueViaInput(ctx context.Context, input FieldsOrderInput) (*FieldsOrderPayload, error) {
	panic("not implemented")
}

// UpdateSomething is the resolver for the updateSomething field.
func (r *mutationResolver) UpdateSomething(ctx context.Context, input SpecialInput) (string, error) {
	panic("not implemented")
}

// UpdatePtrToPtr is the resolver for the updatePtrToPtr field.
func (r *mutationResolver) UpdatePtrToPtr(ctx context.Context, input UpdatePtrToPtrOuter) (*PtrToPtrOuter, error) {
	panic("not implemented")
}

// OldFoo is the resolver for the oldFoo field.
func (r *overlappingFieldsResolver) OldFoo(ctx context.Context, obj *OverlappingFields) (int, error) {
	panic("not implemented")
}

// FieldScalarMarshal is the resolver for the fieldScalarMarshal field.
func (r *panicsResolver) FieldScalarMarshal(ctx context.Context, obj *Panics) ([]MarshalPanic, error) {
	panic("not implemented")
}

// ArgUnmarshal is the resolver for the argUnmarshal field.
func (r *panicsResolver) ArgUnmarshal(ctx context.Context, obj *Panics, u []MarshalPanic) (bool, error) {
	panic("not implemented")
}

// Friends is the resolver for the friends field.
func (r *petResolver) Friends(ctx context.Context, obj *Pet, limit *int) ([]*Pet, error) {
	panic("not implemented")
}

// Value is the resolver for the value field.
func (r *primitiveResolver) Value(ctx context.Context, obj *Primitive) (int, error) {
	panic("not implemented")
}

// Value is the resolver for the value field.
func (r *primitiveStringResolver) Value(ctx context.Context, obj *PrimitiveString) (string, error) {
	panic("not implemented")
}

// Len is the resolver for the len field.
func (r *primitiveStringResolver) Len(ctx context.Context, obj *PrimitiveString) (int, error) {
	panic("not implemented")
}

// InvalidIdentifier is the resolver for the invalidIdentifier field.
func (r *queryResolver) InvalidIdentifier(ctx context.Context) (*invalid_packagename.InvalidIdentifier, error) {
	panic("not implemented")
}

// Collision is the resolver for the collision field.
func (r *queryResolver) Collision(ctx context.Context) (*introspection1.It, error) {
	panic("not implemented")
}

// MapInput is the resolver for the mapInput field.
func (r *queryResolver) MapInput(ctx context.Context, input map[string]interface{}) (*bool, error) {
	panic("not implemented")
}

// Recursive is the resolver for the recursive field.
func (r *queryResolver) Recursive(ctx context.Context, input *RecursiveInputSlice) (*bool, error) {
	panic("not implemented")
}

// NestedInputs is the resolver for the nestedInputs field.
func (r *queryResolver) NestedInputs(ctx context.Context, input [][]*OuterInput) (*bool, error) {
	panic("not implemented")
}

// NestedOutputs is the resolver for the nestedOutputs field.
func (r *queryResolver) NestedOutputs(ctx context.Context) ([][]*OuterObject, error) {
	panic("not implemented")
}

// ModelMethods is the resolver for the modelMethods field.
func (r *queryResolver) ModelMethods(ctx context.Context) (*ModelMethods, error) {
	panic("not implemented")
}

// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, id int) (*User, error) {
	panic("not implemented")
}

// NullableArg is the resolver for the nullableArg field.
func (r *queryResolver) NullableArg(ctx context.Context, arg *int) (*string, error) {
	panic("not implemented")
}

// InputSlice is the resolver for the inputSlice field.
func (r *queryResolver) InputSlice(ctx context.Context, arg []string) (bool, error) {
	panic("not implemented")
}

// InputNullableSlice is the resolver for the inputNullableSlice field.
func (r *queryResolver) InputNullableSlice(ctx context.Context, arg []string) (bool, error) {
	panic("not implemented")
}

// InputOmittable is the resolver for the inputOmittable field.
func (r *queryResolver) InputOmittable(ctx context.Context, arg OmittableInput) (string, error) {
	panic("not implemented")
}

// ShapeUnion is the resolver for the shapeUnion field.
func (r *queryResolver) ShapeUnion(ctx context.Context) (ShapeUnion, error) {
	panic("not implemented")
}

// Autobind is the resolver for the autobind field.
func (r *queryResolver) Autobind(ctx context.Context) (*Autobind, error) {
	panic("not implemented")
}

// DeprecatedField is the resolver for the deprecatedField field.
func (r *queryResolver) DeprecatedField(ctx context.Context) (string, error) {
	panic("not implemented")
}

// Overlapping is the resolver for the overlapping field.
func (r *queryResolver) Overlapping(ctx context.Context) (*OverlappingFields, error) {
	panic("not implemented")
}

// DefaultParameters is the resolver for the defaultParameters field.
func (r *queryResolver) DefaultParameters(ctx context.Context, falsyBoolean *bool, truthyBoolean *bool) (*DefaultParametersMirror, error) {
	panic("not implemented")
}

// DeferCase1 is the resolver for the deferCase1 field.
func (r *queryResolver) DeferCase1(ctx context.Context) (*DeferModel, error) {
	panic("not implemented")
}

// DeferCase2 is the resolver for the deferCase2 field.
func (r *queryResolver) DeferCase2(ctx context.Context) ([]*DeferModel, error) {
	panic("not implemented")
}

// DirectiveArg is the resolver for the directiveArg field.
func (r *queryResolver) DirectiveArg(ctx context.Context, arg string) (*string, error) {
	panic("not implemented")
}

// DirectiveNullableArg is the resolver for the directiveNullableArg field.
func (r *queryResolver) DirectiveNullableArg(ctx context.Context, arg *int, arg2 *int, arg3 *string) (*string, error) {
	panic("not implemented")
}

// DirectiveInputNullable is the resolver for the directiveInputNullable field.
func (r *queryResolver) DirectiveInputNullable(ctx context.Context, arg *InputDirectives) (*string, error) {
	panic("not implemented")
}

// DirectiveInput is the resolver for the directiveInput field.
func (r *queryResolver) DirectiveInput(ctx context.Context, arg InputDirectives) (*string, error) {
	panic("not implemented")
}

// DirectiveInputType is the resolver for the directiveInputType field.
func (r *queryResolver) DirectiveInputType(ctx context.Context, arg InnerInput) (*string, error) {
	panic("not implemented")
}

// DirectiveObject is the resolver for the directiveObject field.
func (r *queryResolver) DirectiveObject(ctx context.Context) (*ObjectDirectives, error) {
	panic("not implemented")
}

// DirectiveObjectWithCustomGoModel is the resolver for the directiveObjectWithCustomGoModel field.
func (r *queryResolver) DirectiveObjectWithCustomGoModel(ctx context.Context) (*ObjectDirectivesWithCustomGoModel, error) {
	panic("not implemented")
}

// DirectiveFieldDef is the resolver for the directiveFieldDef field.
func (r *queryResolver) DirectiveFieldDef(ctx context.Context, ret string) (string, error) {
	panic("not implemented")
}

// DirectiveField is the resolver for the directiveField field.
func (r *queryResolver) DirectiveField(ctx context.Context) (*string, error) {
	panic("not implemented")
}

// DirectiveDouble is the resolver for the directiveDouble field.
func (r *queryResolver) DirectiveDouble(ctx context.Context) (*string, error) {
	panic("not implemented")
}

// DirectiveUnimplemented is the resolver for the directiveUnimplemented field.
func (r *queryResolver) DirectiveUnimplemented(ctx context.Context) (*string, error) {
	panic("not implemented")
}

// EmbeddedCase1 is the resolver for the embeddedCase1 field.
func (r *queryResolver) EmbeddedCase1(ctx context.Context) (*EmbeddedCase1, error) {
	panic("not implemented")
}

// EmbeddedCase2 is the resolver for the embeddedCase2 field.
func (r *queryResolver) EmbeddedCase2(ctx context.Context) (*EmbeddedCase2, error) {
	panic("not implemented")
}

// EmbeddedCase3 is the resolver for the embeddedCase3 field.
func (r *queryResolver) EmbeddedCase3(ctx context.Context) (*EmbeddedCase3, error) {
	panic("not implemented")
}

// EnumInInput is the resolver for the enumInInput field.
func (r *queryResolver) EnumInInput(ctx context.Context, input *InputWithEnumValue) (EnumTest, error) {
	panic("not implemented")
}

// Shapes is the resolver for the shapes field.
func (r *queryResolver) Shapes(ctx context.Context) ([]Shape, error) {
	panic("not implemented")
}

// NoShape is the resolver for the noShape field.
func (r *queryResolver) NoShape(ctx context.Context) (Shape, error) {
	panic("not implemented")
}

// Node is the resolver for the node field.
func (r *queryResolver) Node(ctx context.Context) (Node, error) {
	panic("not implemented")
}

// NoShapeTypedNil is the resolver for the noShapeTypedNil field.
func (r *queryResolver) NoShapeTypedNil(ctx context.Context) (Shape, error) {
	panic("not implemented")
}

// Animal is the resolver for the animal field.
func (r *queryResolver) Animal(ctx context.Context) (Animal, error) {
	panic("not implemented")
}

// NotAnInterface is the resolver for the notAnInterface field.
func (r *queryResolver) NotAnInterface(ctx context.Context) (BackedByInterface, error) {
	panic("not implemented")
}

// Dog is the resolver for the dog field.
func (r *queryResolver) Dog(ctx context.Context) (*Dog, error) {
	panic("not implemented")
}

// Issue896a is the resolver for the issue896a field.
func (r *queryResolver) Issue896a(ctx context.Context) ([]*CheckIssue896, error) {
	panic("not implemented")
}

// MapStringInterface is the resolver for the mapStringInterface field.
func (r *queryResolver) MapStringInterface(ctx context.Context, in map[string]interface{}) (map[string]interface{}, error) {
	panic("not implemented")
}

// MapNestedStringInterface is the resolver for the mapNestedStringInterface field.
func (r *queryResolver) MapNestedStringInterface(ctx context.Context, in *NestedMapInput) (map[string]interface{}, error) {
	panic("not implemented")
}

// ErrorBubble is the resolver for the errorBubble field.
func (r *queryResolver) ErrorBubble(ctx context.Context) (*Error, error) {
	panic("not implemented")
}

// ErrorBubbleList is the resolver for the errorBubbleList field.
func (r *queryResolver) ErrorBubbleList(ctx context.Context) ([]*Error, error) {
	panic("not implemented")
}

// ErrorList is the resolver for the errorList field.
func (r *queryResolver) ErrorList(ctx context.Context) ([]*Error, error) {
	panic("not implemented")
}

// Errors is the resolver for the errors field.
func (r *queryResolver) Errors(ctx context.Context) (*Errors, error) {
	panic("not implemented")
}

// Valid is the resolver for the valid field.
func (r *queryResolver) Valid(ctx context.Context) (string, error) {
	panic("not implemented")
}

// Invalid is the resolver for the invalid field.
func (r *queryResolver) Invalid(ctx context.Context) (string, error) {
	panic("not implemented")
}

// Panics is the resolver for the panics field.
func (r *queryResolver) Panics(ctx context.Context) (*Panics, error) {
	panic("not implemented")
}

// PrimitiveObject is the resolver for the primitiveObject field.
func (r *queryResolver) PrimitiveObject(ctx context.Context) ([]Primitive, error) {
	panic("not implemented")
}

// PrimitiveStringObject is the resolver for the primitiveStringObject field.
func (r *queryResolver) PrimitiveStringObject(ctx context.Context) ([]PrimitiveString, error) {
	panic("not implemented")
}

// PtrToAnyContainer is the resolver for the ptrToAnyContainer field.
func (r *queryResolver) PtrToAnyContainer(ctx context.Context) (*PtrToAnyContainer, error) {
	panic("not implemented")
}

// PtrToSliceContainer is the resolver for the ptrToSliceContainer field.
func (r *queryResolver) PtrToSliceContainer(ctx context.Context) (*PtrToSliceContainer, error) {
	panic("not implemented")
}

// Infinity is the resolver for the infinity field.
func (r *queryResolver) Infinity(ctx context.Context) (float64, error) {
	panic("not implemented")
}

// StringFromContextInterface is the resolver for the stringFromContextInterface field.
func (r *queryResolver) StringFromContextInterface(ctx context.Context) (*StringFromContextInterface, error) {
	panic("not implemented")
}

// StringFromContextFunction is the resolver for the stringFromContextFunction field.
func (r *queryResolver) StringFromContextFunction(ctx context.Context) (string, error) {
	panic("not implemented")
}

// DefaultScalar is the resolver for the defaultScalar field.
func (r *queryResolver) DefaultScalar(ctx context.Context, arg string) (string, error) {
	panic("not implemented")
}

// Slices is the resolver for the slices field.
func (r *queryResolver) Slices(ctx context.Context) (*Slices, error) {
	panic("not implemented")
}

// ScalarSlice is the resolver for the scalarSlice field.
func (r *queryResolver) ScalarSlice(ctx context.Context) ([]byte, error) {
	panic("not implemented")
}

// Fallback is the resolver for the fallback field.
func (r *queryResolver) Fallback(ctx context.Context, arg FallbackToStringEncoding) (FallbackToStringEncoding, error) {
	panic("not implemented")
}

// OptionalUnion is the resolver for the optionalUnion field.
func (r *queryResolver) OptionalUnion(ctx context.Context) (TestUnion, error) {
	panic("not implemented")
}

// VOkCaseValue is the resolver for the vOkCaseValue field.
func (r *queryResolver) VOkCaseValue(ctx context.Context) (*VOkCaseValue, error) {
	panic("not implemented")
}

// VOkCaseNil is the resolver for the vOkCaseNil field.
func (r *queryResolver) VOkCaseNil(ctx context.Context) (*VOkCaseNil, error) {
	panic("not implemented")
}

// ValidType is the resolver for the validType field.
func (r *queryResolver) ValidType(ctx context.Context) (*ValidType, error) {
	panic("not implemented")
}

// VariadicModel is the resolver for the variadicModel field.
func (r *queryResolver) VariadicModel(ctx context.Context) (*VariadicModel, error) {
	panic("not implemented")
}

// WrappedStruct is the resolver for the wrappedStruct field.
func (r *queryResolver) WrappedStruct(ctx context.Context) (*WrappedStruct, error) {
	panic("not implemented")
}

// WrappedScalar is the resolver for the wrappedScalar field.
func (r *queryResolver) WrappedScalar(ctx context.Context) (otherpkg.Scalar, error) {
	panic("not implemented")
}

// WrappedMap is the resolver for the wrappedMap field.
func (r *queryResolver) WrappedMap(ctx context.Context) (WrappedMap, error) {
	panic("not implemented")
}

// WrappedSlice is the resolver for the wrappedSlice field.
func (r *queryResolver) WrappedSlice(ctx context.Context) (WrappedSlice, error) {
	panic("not implemented")
}

// Updated is the resolver for the updated field.
func (r *subscriptionResolver) Updated(ctx context.Context) (<-chan string, error) {
	panic("not implemented")
}

// InitPayload is the resolver for the initPayload field.
func (r *subscriptionResolver) InitPayload(ctx context.Context) (<-chan string, error) {
	panic("not implemented")
}

// DirectiveArg is the resolver for the directiveArg field.
func (r *subscriptionResolver) DirectiveArg(ctx context.Context, arg string) (<-chan *string, error) {
	panic("not implemented")
}

// DirectiveNullableArg is the resolver for the directiveNullableArg field.
func (r *subscriptionResolver) DirectiveNullableArg(ctx context.Context, arg *int, arg2 *int, arg3 *string) (<-chan *string, error) {
	panic("not implemented")
}

// DirectiveDouble is the resolver for the directiveDouble field.
func (r *subscriptionResolver) DirectiveDouble(ctx context.Context) (<-chan *string, error) {
	panic("not implemented")
}

// DirectiveUnimplemented is the resolver for the directiveUnimplemented field.
func (r *subscriptionResolver) DirectiveUnimplemented(ctx context.Context) (<-chan *string, error) {
	panic("not implemented")
}

// Issue896b is the resolver for the issue896b field.
func (r *subscriptionResolver) Issue896b(ctx context.Context) (<-chan []*CheckIssue896, error) {
	panic("not implemented")
}

// ErrorRequired is the resolver for the errorRequired field.
func (r *subscriptionResolver) ErrorRequired(ctx context.Context) (<-chan *Error, error) {
	panic("not implemented")
}

// Friends is the resolver for the friends field.
func (r *userResolver) Friends(ctx context.Context, obj *User) ([]*User, error) {
	panic("not implemented")
}

// Pets is the resolver for the pets field.
func (r *userResolver) Pets(ctx context.Context, obj *User, limit *int) ([]*Pet, error) {
	panic("not implemented")
}

// Get is the resolver for the get field.
func (r *wrappedMapResolver) Get(ctx context.Context, obj WrappedMap, key string) (string, error) {
	panic("not implemented")
}

// Get is the resolver for the get field.
func (r *wrappedSliceResolver) Get(ctx context.Context, obj WrappedSlice, idx int) (string, error) {
	panic("not implemented")
}

// BackedByInterface returns BackedByInterfaceResolver implementation.
func (r *Resolver) BackedByInterface() BackedByInterfaceResolver {
	return &backedByInterfaceResolver{r}
}

// DeferModel returns DeferModelResolver implementation.
func (r *Resolver) DeferModel() DeferModelResolver { return &deferModelResolver{r} }

// Errors returns ErrorsResolver implementation.
func (r *Resolver) Errors() ErrorsResolver { return &errorsResolver{r} }

// ForcedResolver returns ForcedResolverResolver implementation.
func (r *Resolver) ForcedResolver() ForcedResolverResolver { return &forcedResolverResolver{r} }

// ModelMethods returns ModelMethodsResolver implementation.
func (r *Resolver) ModelMethods() ModelMethodsResolver { return &modelMethodsResolver{r} }

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// OverlappingFields returns OverlappingFieldsResolver implementation.
func (r *Resolver) OverlappingFields() OverlappingFieldsResolver {
	return &overlappingFieldsResolver{r}
}

// Panics returns PanicsResolver implementation.
func (r *Resolver) Panics() PanicsResolver { return &panicsResolver{r} }

// Pet returns PetResolver implementation.
func (r *Resolver) Pet() PetResolver { return &petResolver{r} }

// Primitive returns PrimitiveResolver implementation.
func (r *Resolver) Primitive() PrimitiveResolver { return &primitiveResolver{r} }

// PrimitiveString returns PrimitiveStringResolver implementation.
func (r *Resolver) PrimitiveString() PrimitiveStringResolver { return &primitiveStringResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Subscription returns SubscriptionResolver implementation.
func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

// User returns UserResolver implementation.
func (r *Resolver) User() UserResolver { return &userResolver{r} }

// WrappedMap returns WrappedMapResolver implementation.
func (r *Resolver) WrappedMap() WrappedMapResolver { return &wrappedMapResolver{r} }

// WrappedSlice returns WrappedSliceResolver implementation.
func (r *Resolver) WrappedSlice() WrappedSliceResolver { return &wrappedSliceResolver{r} }

type backedByInterfaceResolver struct{ *Resolver }
type deferModelResolver struct{ *Resolver }
type errorsResolver struct{ *Resolver }
type forcedResolverResolver struct{ *Resolver }
type modelMethodsResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type overlappingFieldsResolver struct{ *Resolver }
type panicsResolver struct{ *Resolver }
type petResolver struct{ *Resolver }
type primitiveResolver struct{ *Resolver }
type primitiveStringResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
type userResolver struct{ *Resolver }
type wrappedMapResolver struct{ *Resolver }
type wrappedSliceResolver struct{ *Resolver }
