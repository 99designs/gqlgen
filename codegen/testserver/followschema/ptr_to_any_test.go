package followschema

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
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
