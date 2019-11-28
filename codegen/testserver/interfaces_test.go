package testserver

import (
	"context"
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

func TestInterfaces(t *testing.T) {
	t.Run("slices of interfaces are not pointers", func(t *testing.T) {
		field, ok := reflect.TypeOf((*QueryResolver)(nil)).Elem().MethodByName("Shapes")
		require.True(t, ok)
		require.Equal(t, "[]testserver.Shape", field.Type.Out(0).String())
	})

	t.Run("interfaces can be nil", func(t *testing.T) {
		resolvers := &Stub{}
		resolvers.QueryResolver.NoShape = func(ctx context.Context) (shapes Shape, e error) {
			return nil, nil
		}

		srv := handler.NewDefaultServer(
			NewExecutableSchema(Config{
				Resolvers: resolvers,
				Directives: DirectiveRoot{
					MakeNil: func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
						return nil, nil
					},
				},
			}),
		)

		c := client.New(srv)

		var resp interface{}
		c.MustPost(`{ noShape { area } }`, &resp)
	})

	t.Run("interfaces can be typed nil", func(t *testing.T) {
		resolvers := &Stub{}
		resolvers.QueryResolver.NoShapeTypedNil = func(ctx context.Context) (shapes Shape, e error) {
			panic("should not be called")
		}

		srv := handler.NewDefaultServer(
			NewExecutableSchema(Config{
				Resolvers: resolvers,
				Directives: DirectiveRoot{
					MakeTypedNil: func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
						var circle *Circle
						return circle, nil
					},
				},
			}),
		)

		c := client.New(srv)

		var resp interface{}
		c.MustPost(`{ noShapeTypedNil { area } }`, &resp)
	})

	t.Run("interfaces can be nil (test with code-generated resolver)", func(t *testing.T) {
		resolvers := &Stub{}
		resolvers.QueryResolver.Animal = func(ctx context.Context) (animal Animal, e error) {
			panic("should not be called")
		}

		srv := handler.NewDefaultServer(
			NewExecutableSchema(Config{
				Resolvers: resolvers,
				Directives: DirectiveRoot{
					MakeTypedNil: func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
						var dog *Dog // return a typed nil, not just nil
						return dog, nil
					},
				},
			}),
		)

		c := client.New(srv)

		var resp interface{}
		c.MustPost(`{ animal { species } }`, &resp)
	})
}
