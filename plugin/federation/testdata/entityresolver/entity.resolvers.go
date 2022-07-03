package entityresolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver/generated"
)

// FindHelloByName is the resolver for the findHelloByName field.
func (r *entityResolver) FindHelloByName(ctx context.Context, name string) (*generated.Hello, error) {
	return &generated.Hello{
		Name: name,
	}, nil
}

// FindHelloMultiSingleKeysByKey1AndKey2 is the resolver for the findHelloMultiSingleKeysByKey1AndKey2 field.
func (r *entityResolver) FindHelloMultiSingleKeysByKey1AndKey2(ctx context.Context, key1 string, key2 string) (*generated.HelloMultiSingleKeys, error) {
	panic(fmt.Errorf("not implemented"))
}

// FindHelloWithErrorsByName is the resolver for the findHelloWithErrorsByName field.
func (r *entityResolver) FindHelloWithErrorsByName(ctx context.Context, name string) (*generated.HelloWithErrors, error) {
	if name == "inject error" {
		return nil, generated.ErrResolvingHelloWithErrorsByName
	} else if name == "" {
		return nil, generated.ErrEmptyKeyResolvingHelloWithErrorsByName
	}

	return &generated.HelloWithErrors{
		Name: name,
	}, nil
}

// FindManyMultiHelloByNames is the resolver for the findManyMultiHelloByNames field.
func (r *entityResolver) FindManyMultiHelloByNames(ctx context.Context, reps []*generated.MultiHelloByNamesInput) ([]*generated.MultiHello, error) {
	results := []*generated.MultiHello{}

	for _, item := range reps {
		results = append(results, &generated.MultiHello{
			Name: item.Name + " - from multiget",
		})
	}

	return results, nil
}

// FindManyMultiHelloMultipleRequiresByNames is the resolver for the findManyMultiHelloMultipleRequiresByNames field.
func (r *entityResolver) FindManyMultiHelloMultipleRequiresByNames(ctx context.Context, reps []*generated.MultiHelloMultipleRequiresByNamesInput) ([]*generated.MultiHelloMultipleRequires, error) {
	results := make([]*generated.MultiHelloMultipleRequires, len(reps))

	for i := range reps {
		results[i] = &generated.MultiHelloMultipleRequires{
			Name: reps[i].Name,
		}
	}

	return results, nil
}

// FindManyMultiHelloRequiresByNames is the resolver for the findManyMultiHelloRequiresByNames field.
func (r *entityResolver) FindManyMultiHelloRequiresByNames(ctx context.Context, reps []*generated.MultiHelloRequiresByNamesInput) ([]*generated.MultiHelloRequires, error) {
	results := make([]*generated.MultiHelloRequires, len(reps))

	for i := range reps {
		results[i] = &generated.MultiHelloRequires{
			Name: reps[i].Name,
		}
	}

	return results, nil
}

// FindManyMultiHelloWithErrorByNames is the resolver for the findManyMultiHelloWithErrorByNames field.
func (r *entityResolver) FindManyMultiHelloWithErrorByNames(ctx context.Context, reps []*generated.MultiHelloWithErrorByNamesInput) ([]*generated.MultiHelloWithError, error) {
	return nil, fmt.Errorf("error resolving MultiHelloWorldWithError")
}

// FindManyMultiPlanetRequiresNestedByNames is the resolver for the findManyMultiPlanetRequiresNestedByNames field.
func (r *entityResolver) FindManyMultiPlanetRequiresNestedByNames(ctx context.Context, reps []*generated.MultiPlanetRequiresNestedByNamesInput) ([]*generated.MultiPlanetRequiresNested, error) {
	worlds := map[string]*generated.World{
		"earth": {
			Foo: "A",
		},
		"mars": {
			Foo: "B",
		},
	}

	results := make([]*generated.MultiPlanetRequiresNested, len(reps))

	for i := range reps {
		name := reps[i].Name
		world, ok := worlds[name]
		if !ok {
			return nil, fmt.Errorf("unknown planet: %s", name)
		}

		results[i] = &generated.MultiPlanetRequiresNested{
			Name:  name,
			World: world,
		}
	}

	return results, nil
}

// FindPlanetMultipleRequiresByName is the resolver for the findPlanetMultipleRequiresByName field.
func (r *entityResolver) FindPlanetMultipleRequiresByName(ctx context.Context, name string) (*generated.PlanetMultipleRequires, error) {
	return &generated.PlanetMultipleRequires{Name: name}, nil
}

// FindPlanetRequiresByName is the resolver for the findPlanetRequiresByName field.
func (r *entityResolver) FindPlanetRequiresByName(ctx context.Context, name string) (*generated.PlanetRequires, error) {
	return &generated.PlanetRequires{
		Name: name,
	}, nil
}

// FindPlanetRequiresNestedByName is the resolver for the findPlanetRequiresNestedByName field.
func (r *entityResolver) FindPlanetRequiresNestedByName(ctx context.Context, name string) (*generated.PlanetRequiresNested, error) {
	worlds := map[string]*generated.World{
		"earth": {
			Foo: "A",
		},
		"mars": {
			Foo: "B",
		},
	}
	world, ok := worlds[name]
	if !ok {
		return nil, fmt.Errorf("unknown planet: %s", name)
	}

	return &generated.PlanetRequiresNested{
		Name:  name,
		World: world,
	}, nil
}

// FindWorldByHelloNameAndFoo is the resolver for the findWorldByHelloNameAndFoo field.
func (r *entityResolver) FindWorldByHelloNameAndFoo(ctx context.Context, helloName string, foo string) (*generated.World, error) {
	return &generated.World{
		Hello: &generated.Hello{
			Name: helloName,
		},
		Foo: foo,
	}, nil
}

// FindWorldNameByName is the resolver for the findWorldNameByName field.
func (r *entityResolver) FindWorldNameByName(ctx context.Context, name string) (*generated.WorldName, error) {
	return &generated.WorldName{
		Name: name,
	}, nil
}

// FindWorldWithMultipleKeysByHelloNameAndFoo is the resolver for the findWorldWithMultipleKeysByHelloNameAndFoo field.
func (r *entityResolver) FindWorldWithMultipleKeysByHelloNameAndFoo(ctx context.Context, helloName string, foo string) (*generated.WorldWithMultipleKeys, error) {
	return &generated.WorldWithMultipleKeys{
		Hello: &generated.Hello{
			Name: helloName,
		},
		Foo: foo,
	}, nil
}

// FindWorldWithMultipleKeysByBar is the resolver for the findWorldWithMultipleKeysByBar field.
func (r *entityResolver) FindWorldWithMultipleKeysByBar(ctx context.Context, bar int) (*generated.WorldWithMultipleKeys, error) {
	return &generated.WorldWithMultipleKeys{
		Bar: bar,
	}, nil
}

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
