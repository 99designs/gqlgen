package followschema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestPtrToAny(t *testing.T) {
	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	var a any = `{"some":"thing"}`
	resolvers.QueryResolver.PtrToAnyContainer = func(ctx context.Context) (wrappedStruct *PtrToAnyContainer, e error) {
		ptrToAnyContainer := PtrToAnyContainer{
			PtrToAny: &a,
		}
		return &ptrToAnyContainer, nil
	}

	t.Run("binding to pointer to any", func(t *testing.T) {
		var resp struct {
			PtrToAnyContainer struct {
				Binding *any
			}
		}

		err := c.Post(`query { ptrToAnyContainer { binding }}`, &resp)
		require.NoError(t, err)

		require.Equal(t, &a, resp.PtrToAnyContainer.Binding)
	})
}

func TestPtrToAny_Null(t *testing.T) {
	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	srv.SetRecoverFunc(nil)
	c := client.New(srv)

	resolvers.QueryResolver.PtrToAnyContainer = func(ctx context.Context) (wrappedStruct *PtrToAnyContainer, e error) {
		return &PtrToAnyContainer{PtrToAny: nil}, nil
	}

	t.Run("nil pointer to any should return null without panic", func(t *testing.T) {
		var resp struct {
			PtrToAnyContainer struct {
				Binding *any
			}
		}

		require.NotPanics(t, func() {
			err := c.Post(`query { ptrToAnyContainer { binding }}`, &resp)
			require.NoError(t, err)
			require.Nil(t, resp.PtrToAnyContainer.Binding)
		})
	})
}
