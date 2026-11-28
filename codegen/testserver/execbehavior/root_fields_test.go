package execbehavior

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

// rootFieldLog records the order in which the middleware and resolvers of one server run.
type rootFieldLog struct {
	mu      sync.Mutex
	entries []string
}

func (l *rootFieldLog) record(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, s)
}

// newRootFieldServer returns a server whose root field and field middleware, and whose
// resolvers, record into the returned log.
func newRootFieldServer() (*handler.Server, *rootFieldLog) {
	l := &rootFieldLog{}
	resolvers := &Stub{}
	resolvers.QueryResolver.Ping = func(ctx context.Context) (string, error) {
		fc := graphql.GetFieldContext(ctx)
		l.record("resolve " + fc.Field.Name + " under " + fc.Parent.Object)
		return "pong", nil
	}
	var appended []string
	resolvers.MutationResolver.AppendLog = func(ctx context.Context, entry string) ([]string, error) {
		l.record("resolve appendLog " + entry)
		appended = append(appended, entry)
		return appended, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})
	srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
		rc := graphql.GetRootFieldContext(ctx)
		l.record("root " + rc.Object + "=" + rc.Field.Alias)
		if rc.Field.Alias == "replaced" {
			return graphql.MarshalString("replaced by the root middleware")
		}
		return next(ctx)
	})
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
		fc := graphql.GetFieldContext(ctx)
		if fc.Parent.Object == "Query" || fc.Parent.Object == "Mutation" {
			l.record("field " + fc.Field.Alias)
		}
		return next(ctx)
	})
	return srv, l
}

func TestRootFieldMiddleware(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		srv, l := newRootFieldServer()
		got := post(t, srv, `{ __typename ping replaced: ping }`)
		log := l.entries
		require.JSONEq(
			t,
			`{"data":{"__typename":"Query","ping":"pong","replaced":"replaced by the root middleware"}}`,
			got,
		)
		// Query fields resolve concurrently, so only the order within a field is fixed:
		// the root middleware wraps the field middleware, and replacing the result skips
		// the field. __typename does not run the middleware.
		require.ElementsMatch(t, []string{
			"root ping=ping", "field ping", "resolve ping under Query",
			"root ping=replaced",
		}, log)
		require.Less(t, slices.Index(log, "root ping=ping"), slices.Index(log, "field ping"))
		require.Less(
			t,
			slices.Index(log, "field ping"),
			slices.Index(log, "resolve ping under Query"),
		)
	})

	t.Run("mutation", func(t *testing.T) {
		// Mutation fields resolve one after the other, in the order of the query.
		srv, l := newRootFieldServer()
		got := post(
			t,
			srv,
			`mutation { first: appendLog(entry: "a") second: appendLog(entry: "b") }`,
		)
		require.JSONEq(t, `{"data":{"first":["a"],"second":["a","b"]}}`, got)
		require.Equal(t, []string{
			"root appendLog=first", "field first", "resolve appendLog a",
			"root appendLog=second", "field second", "resolve appendLog b",
		}, l.entries)
	})
}
