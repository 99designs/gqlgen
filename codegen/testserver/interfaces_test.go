package testserver

import (
	"context"
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
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

		srv := handler.GraphQL(
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
}
