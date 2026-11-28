package execbehavior

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestMapInputFieldFromDirective(t *testing.T) {
	resolvers := &Stub{}
	// received describes input["note"] as the resolver sees it.
	var received string
	resolvers.QueryResolver.MapNote = func(ctx context.Context, input map[string]any) (*string, error) {
		v, ok := input["note"]
		switch {
		case !ok:
			received = "absent"
		case v == nil:
			received = "nil"
		default:
			received = fmt.Sprintf("%T", v)
		}
		return nil, nil
	}
	srv := newServer(resolvers, DirectiveRoot{
		ReturnValue: func(
			ctx context.Context,
			obj any,
			next graphql.Resolver,
			kind string,
		) (any, error) {
			if _, err := next(ctx); err != nil {
				return nil, err
			}
			return nil, nil
		},
	})

	got := post(t, srv, `{ mapNote(input: { note: "n" }) }`)
	require.JSONEq(t, `{"data":{"mapNote":null}}`, got)
	// The nil of the directive is stored as it is, so that input["note"] == nil holds, as
	// for an input bound to a struct.
	require.Equal(t, "nil", received)

	got = post(t, srv, `{ mapNote(input: {}) }`)
	require.JSONEq(t, `{"data":{"mapNote":null}}`, got)
	require.Equal(t, "absent", received)
}
