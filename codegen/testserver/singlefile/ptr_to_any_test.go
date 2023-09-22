package singlefile

import (
	"context"
	"testing"

	"github.com/apito-cms/gqlgen/client"
	"github.com/apito-cms/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

func TestPtrToAny(t *testing.T) {
	resolvers := &Stub{}

	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers})))

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
