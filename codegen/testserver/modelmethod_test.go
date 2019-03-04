package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestModelMethods(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.ModelMethods = func(ctx context.Context) (methods *ModelMethods, e error) {
		return &ModelMethods{}, nil
	}
	resolver.ModelMethodsResolver.ResolverField = func(ctx context.Context, obj *ModelMethods) (b bool, e error) {
		return true, nil
	}

	srv := httptest.NewServer(
		handler.GraphQL(
			NewExecutableSchema(Config{Resolvers: resolver}),
		))
	defer srv.Close()
	c := client.New(srv.URL)
	t.Run("without context", func(t *testing.T) {
		var resp struct {
			ModelMethods struct {
				NoContext bool
			}
		}
		err := c.Post(`query { modelMethods{ noContext } }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.ModelMethods.NoContext)
	})
	t.Run("with context", func(t *testing.T) {
		var resp struct {
			ModelMethods struct {
				WithContext bool
			}
		}
		err := c.Post(`query { modelMethods{ withContext } }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.ModelMethods.WithContext)
	})
}
