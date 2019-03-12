package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestMaps(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.MapStringInterface = func(ctx context.Context, in map[string]interface{}) (i map[string]interface{}, e error) {
		return in, nil
	}

	srv := httptest.NewServer(
		handler.GraphQL(
			NewExecutableSchema(Config{Resolvers: resolver}),
		))
	defer srv.Close()
	c := client.New(srv.URL)
	t.Run("unset", func(t *testing.T) {
		var resp struct {
			MapStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapStringInterface { a, b } }`, &resp)
		require.NoError(t, err)
		require.Nil(t, resp.MapStringInterface)
	})

	t.Run("nil", func(t *testing.T) {
		var resp struct {
			MapStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapStringInterface(in: null) { a, b } }`, &resp)
		require.NoError(t, err)
		require.Nil(t, resp.MapStringInterface)
	})

	t.Run("values", func(t *testing.T) {
		var resp struct {
			MapStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapStringInterface(in: { a: "a", b: null }) { a, b } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "a", resp.MapStringInterface["a"])
		require.Nil(t, resp.MapStringInterface["b"])
	})
}
