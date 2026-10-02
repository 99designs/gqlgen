package execbehavior

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestInputObjectDirective(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.CheckedInput = func(ctx context.Context, input CheckedInput) (string, error) {
		return input.Mode + ":" + input.Value, nil
	}
	// newCheckServer returns a server whose inputCheck acts on the mode of the input, and
	// stores the raw input it receives in received, unless received is nil.
	newCheckServer := func(received *any) *handler.Server {
		return newServer(resolvers, DirectiveRoot{
			InputCheck: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				if received != nil {
					*received = obj
				}
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}
				switch obj.(map[string]any)["mode"] {
				case "error":
					return nil, errors.New("rejected by inputCheck")
				case "replace":
					in := res.(CheckedInput)
					in.Value = "replaced " + in.Value
					return in, nil
				case "wrong":
					return "not an input", nil
				case "nil":
					return nil, nil
				}
				return res, nil
			},
		})
	}
	srv := newCheckServer(nil)

	t.Run("passes the input on", func(t *testing.T) {
		var received any
		srv := newCheckServer(&received)
		got := post(t, srv, `{ checkedInput(input: { mode: "pass", value: "v" }) }`)
		require.JSONEq(t, `{"data":{"checkedInput":"pass:v"}}`, got)
		require.Equal(t, map[string]any{"mode": "pass", "value": "v"}, received)
	})

	t.Run("replaces the input", func(t *testing.T) {
		got := post(t, srv, `{ checkedInput(input: { mode: "replace", value: "v" }) }`)
		require.JSONEq(t, `{"data":{"checkedInput":"replace:replaced v"}}`, got)
	})

	t.Run("fails", func(t *testing.T) {
		got := post(t, srv, `{ checkedInput(input: { mode: "error", value: "v" }) }`)
		require.JSONEq(
			t,
			`{"data":null,"errors":[{"message":"rejected by inputCheck","path":["checkedInput","input"]}]}`,
			got,
		)
	})

	t.Run("returns another type", func(t *testing.T) {
		got := post(t, srv, `{ checkedInput(input: { mode: "wrong", value: "v" }) }`)
		require.JSONEq(
			t,
			`{"data":null,"errors":[{"message":"unexpected type string from INPUT_OBJECT directive, should be CheckedInput","path":["checkedInput","input"]}]}`,
			got,
		)
	})

	t.Run("returns nil", func(t *testing.T) {
		got := post(t, srv, `{ checkedInput(input: { mode: "nil", value: "v" }) }`)
		require.JSONEq(
			t,
			`{"data":null,"errors":[{"message":"unexpected type <nil> from INPUT_OBJECT directive, should be CheckedInput","path":["checkedInput","input"]}]}`,
			got,
		)
	})
}
