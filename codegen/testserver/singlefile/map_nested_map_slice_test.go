package singlefile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestMapNestedMapSlice(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.MapNestedMapSlice = func(ctx context.Context, input map[string]any) (*bool, error) {
		require.NotNil(t, input, "expected input")
		require.NotNil(t, input["recurse"], "expected recurse")
		require.IsType(
			t,
			[]map[string]any{},
			input["recurse"],
			"expected recurse as [][]map[string]any",
		)
		recurse := input["recurse"].([]map[string]any)
		require.Len(t, recurse, 1, "expected 1 item in recurse")
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
