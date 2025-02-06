package followschema

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
)

func TestErrorInsideMutationArgument(t *testing.T) {
	resolvers := &Stub{}
	resolvers.MutationResolver.UpdateSomething = func(_ context.Context, input SpecialInput) (s string, err error) {
		return "Hello world", nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("mutation with correct input doesn't return error", func(t *testing.T) {
		var resp map[string]any
		input := map[string]any{
			"nesting": map[string]any{
				"field": "email@example.com",
			},
		}
		err := c.Post(
			`mutation TestMutation($input: SpecialInput!) { updateSomething(input: $input) }`,
			&resp,
			client.Var("input", input),
		)
		require.Equal(t, "Hello world", resp["updateSomething"])
		require.NoError(t, err)
	})

	t.Run("mutation with incorrect input returns full path", func(t *testing.T) {
		var resp map[string]any
		input := map[string]any{
			"nesting": map[string]any{
				"field": "not-an-email",
			},
		}
		err := c.Post(
			`mutation TestMutation($input: SpecialInput!) { updateSomething(input: $input) }`,
			&resp,
			client.Var("input", input),
		)
		require.EqualError(t, err, `[{"message":"invalid email format","path":["updateSomething","input","nesting","field"]}]`)
	})
}
