package execbehavior

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestSchemaValues(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.DefaultMap = func(ctx context.Context, input DefaultMapInput) (*string, error) {
		s := fmt.Sprint(input.Meta)
		input.Meta["b"] = 2
		return &s, nil
	}
	srv := newServer(resolvers, DirectiveRoot{
		AnyArg: func(ctx context.Context, obj any, next graphql.Resolver, value any) (any, error) {
			s := fmt.Sprint(value)
			if m, ok := value.(map[string]any); ok {
				m["b"] = 2
			}
			return &s, nil
		},
	})

	// The resolver and the directive change the maps they get, which does not change
	// the values of the next request.
	query := `{ anyArgAbsent anyArgNull anyArgMap defaultMap(input: {}) }`
	want := `{"data":{"anyArgAbsent":"<nil>","anyArgNull":"<nil>","anyArgMap":"map[a:1]",` +
		`"defaultMap":"map[a:1]"}}`
	require.JSONEq(t, want, post(t, srv, query))
	require.JSONEq(t, want, post(t, srv, query))
}
