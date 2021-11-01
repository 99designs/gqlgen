package entityresolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver/generated"
)

func (r *entityResolver) FindHelloByName(ctx context.Context, name string) (*generated.Hello, error) {
	return &generated.Hello{
		Name: name,
	}, nil
}

func (r *entityResolver) FindHelloWithErrorsByName(ctx context.Context, name string) (*generated.HelloWithErrors, error) {
	if name == "inject error" {
		return nil, fmt.Errorf("error resolving HelloWithErrorsByName")
	} else if name == "" {
		return nil, fmt.Errorf("error (empty key) resolving HelloWithErrorsByName")
	}

	return &generated.HelloWithErrors{
		Name: name,
	}, nil
}

func (r *entityResolver) FindManyMultiHellosByName(ctx context.Context, reps []*generated.EntityResolverfindManyMultiHellosByNameInput) ([]*generated.MultiHello, error) {
	results := []*generated.MultiHello{}

	for _, item := range reps {
		results = append(results, &generated.MultiHello{
			Name: item.Name + " - from multiget",
		})
	}

	return results, nil
}

func (r *entityResolver) FindManyMultiHelloWithErrorsByName(ctx context.Context, reps []*generated.EntityResolverfindManyMultiHelloWithErrorsByNameInput) ([]*generated.MultiHelloWithError, error) {
	return nil, fmt.Errorf("error resolving MultiHelloWorldWithError")
}

func (r *entityResolver) FindPlanetRequiresByName(ctx context.Context, name string) (*generated.PlanetRequires, error) {
	return &generated.PlanetRequires{
		Name: name,
	}, nil
}

func (r *entityResolver) FindWorldByHelloNameAndFoo(ctx context.Context, helloName string, foo string) (*generated.World, error) {
	return &generated.World{
		Hello: &generated.Hello{
			Name: helloName,
		},
		Foo: foo,
	}, nil
}

func (r *entityResolver) FindWorldNameByName(ctx context.Context, name string) (*generated.WorldName, error) {
	return &generated.WorldName{
		Name: name,
	}, nil
}

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
