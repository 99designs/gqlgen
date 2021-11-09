package entityresolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver/generated"
)

func (r *entityResolver) FindHelloByName(ctx context.Context, name string) (*generated.Hello, error) {
	return &generated.Hello{
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

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
