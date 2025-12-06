package singlefile

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestMapNestedMapSlice(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.MapNestedMapSlice = func(ctx context.Context, input map[string]interface{}) (*bool, error) {
		return nil, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("recursive input", func(t *testing.T) {
		var resp struct {
			MapNestedMapSlice bool
		}
		// recurse is [MapNestedMapSlice!]
		err := c.Post(
			`query { mapNestedMapSlice(input: { recurse: [{ name: "child" }] }) }`,
			&resp,
		)
		require.NoError(t, err)
	})
}
