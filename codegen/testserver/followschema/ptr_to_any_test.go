package followschema

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
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

	t.Run("pointer to any", func(t *testing.T) {
		var resp struct {
			PtrToAnyContainer struct {
				PtrToAny *any
			}
		}

		err := c.Post(`query { ptrToAnyContainer {  ptrToAny }}`, &resp)
		require.NoError(t, err)

		require.Equal(t, &a, resp.PtrToAnyContainer.PtrToAny)
	})
}
