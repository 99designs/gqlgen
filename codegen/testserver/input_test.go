package testserver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

func TestInput(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	c := client.New(srv)

	t.Run("when input slice nullable", func(t *testing.T) {
		resolvers.QueryResolver.InputNullableSlice = func(ctx context.Context, arg []string) (b bool, e error) {
			return arg == nil, nil
		}

		var resp struct {
			InputNullableSlice bool
		}
		var err error
		err = c.Post(`query { inputNullableSlice(arg: null) }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.InputNullableSlice)

		err = c.Post(`query { inputNullableSlice(arg: []) }`, &resp)
		require.NoError(t, err)
		require.False(t, resp.InputNullableSlice)
	})

	t.Run("coerce single value to slice", func(t *testing.T) {
		check := func(ctx context.Context, arg []string) (b bool, e error) {
			return len(arg) == 1 && arg[0] == "coerced", nil
		}
		resolvers.QueryResolver.InputSlice = check
		resolvers.QueryResolver.InputNullableSlice = check

		var resp struct {
			Coerced bool
		}
		var err error
		err = c.Post(`query { coerced: inputSlice(arg: "coerced") }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.Coerced)

		err = c.Post(`query { coerced: inputNullableSlice(arg: "coerced") }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.Coerced)
	})
}
