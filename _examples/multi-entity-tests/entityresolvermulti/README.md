# Entity resolver multi + computed requires

This approach uses the following gqlgen config:

```yml
# Uncomment to enable federation
federation:
  filename: graph/federation.go
  package: graph
  version: 2
  options:
   computed_requires: true
   entity_resolver_multi: true
```

The issue with this approach is that the initial entity resolver:

```go
// FindManyProductByIDs is the resolver for the findManyProductByIDs field.
func (r *entityResolver) FindManyProductByIDs(ctx context.Context, reps []*model.ProductByIDsInput) ([]*model.Product, error) {
	panic(fmt.Errorf("not implemented: FindManyProductByIDs - findManyProductByIDs"))
}
```

has no access to the @required fields. It is only available to the field resolver

```go
// Display is the resolver for the display field.
func (r *productResolver) Display(ctx context.Context, obj *model.Product, federationRequires map[string]any) (*model.Variation, error) {
	panic(fmt.Errorf("not implemented: Display - display"))
}
```

which means that there's no function scope in which we get access to all the @required variations at once to make one ML model call.

If there's no intensive computations in the field resolver function this is a fine approach. However we are trying to eliminate the N+1 problem
so this does not satisfy our requirements.
