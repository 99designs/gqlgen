package execbehavior

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestFieldReturningRootType(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Ping = func(ctx context.Context) (string, error) {
		return "pong", nil
	}
	resolvers.QueryResolver.Viewer = func(ctx context.Context) (*Viewer, error) {
		return &Viewer{Name: "viewer"}, nil
	}
	resolvers.QueryResolver.ValueViewer = func(ctx context.Context) (*ValueViewer, error) {
		return &ValueViewer{Name: "value viewer"}, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})
	var (
		mu     sync.Mutex
		fields []string
	)
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
		fc := graphql.GetFieldContext(ctx)
		mu.Lock()
		fields = append(fields, fc.Object+"."+fc.Field.Name)
		mu.Unlock()
		return next(ctx)
	})

	got := post(t, srv, `{
		viewer {
			name
			query { ping viewer { name } }
			optionalQuery { ping }
			mutation { __typename }
		}
		valueViewer { query { ping } }
	}`)
	require.JSONEq(t, `{"data":{"viewer":{
		"name":"viewer",
		"query":{"ping":"pong","viewer":{"name":"viewer"}},
		"optionalQuery":{"ping":"pong"},
		"mutation":{"__typename":"Mutation"}
	},"valueViewer":{"query":{"ping":"pong"}}}}`, got)

	// The fields returning a root type do not run the field middleware; the fields
	// selected under them do.
	require.ElementsMatch(t, []string{
		"Query.viewer", "Viewer.name",
		"Query.ping", "Query.viewer", "Viewer.name",
		"Query.ping",
		"Query.valueViewer", "Query.ping",
	}, fields)
}
