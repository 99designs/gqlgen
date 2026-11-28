package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// complexityProbe records what the complexity functions and directives of one server
// were called with.
type complexityProbe struct {
	gotDefault        *int
	directiveCalls    int
	wrongTypeArgCalls int
}

// newComplexityServer returns a server for resolvers with the complexity functions of the
// tests and the given complexity limit, and the probe that the server records into.
func newComplexityServer(resolvers *Stub, limit int) (*handler.Server, *complexityProbe) {
	p := &complexityProbe{}
	cfg := Config{Resolvers: resolvers}
	cfg.Complexity.Query.LimitedItems = func(childComplexity, count int) int {
		return count * childComplexity
	}
	cfg.Complexity.Query.ArgProbe = func(childComplexity int, strict *Strict, withDefault *int, plain *string) int {
		p.gotDefault = withDefault
		return 100
	}
	cfg.Complexity.ArgObject.Echo = func(childComplexity int, strict Strict) int {
		return 50
	}
	cfg.Directives.ReturnValue = func(ctx context.Context, obj any, next graphql.Resolver, kind string) (any, error) {
		p.directiveCalls++
		return next(ctx)
	}
	cfg.Complexity.Query.WrongTypeArg = func(childComplexity int, value *string) int {
		p.wrongTypeArgCalls++
		return 1000
	}
	srv := handler.New(NewExecutableSchema(cfg))
	srv.AddTransport(transport.POST{})
	srv.Use(extension.FixedComplexityLimit(limit))
	return srv, p
}

// TestComplexity has no schema of its own: it uses the fields of root_typed_field.graphql,
// worker_limit.graphql, arguments.graphql and wrong_type.graphql.
func TestComplexity(t *testing.T) {
	// complexity sends query to a new server with a limit of 1, with which every
	// operation fails and the error says its complexity.
	complexity := func(t *testing.T, query string) (string, *complexityProbe) {
		t.Helper()
		srv, p := newComplexityServer(&Stub{}, 1)
		return post(t, srv, query), p
	}
	exceeds := func(n string) string {
		return `{"data":null,"errors":[{"message":"operation has complexity ` + n +
			`, which exceeds the limit of 1","extensions":{"code":"COMPLEXITY_LIMIT_EXCEEDED"}}]}`
	}

	t.Run("default complexity counts each field", func(t *testing.T) {
		got, _ := complexity(t, `{ viewer { name query { ping } } }`)
		require.JSONEq(t, exceeds("4"), got)
	})

	t.Run("complexity function with an argument", func(t *testing.T) {
		got, _ := complexity(t, `{ limitedItems(count: 8) { id slow } }`)
		require.JSONEq(t, exceeds("16"), got)
	})

	t.Run("complexity function receives argument defaults", func(t *testing.T) {
		got, p := complexity(t, `{ argProbe }`)
		require.JSONEq(t, exceeds("100"), got)
		require.NotNil(t, p.gotDefault)
		require.Equal(t, 7, *p.gotDefault)
	})

	t.Run("nested complexity function", func(t *testing.T) {
		got, _ := complexity(t, `{ argObject { ok echo(strict: "s") } }`)
		require.JSONEq(t, exceeds("52"), got)
	})

	t.Run("arguments that fail to unmarshal fall back to the default", func(t *testing.T) {
		got, _ := complexity(t, `{ argObject { ok echo(strict: "bad") } }`)
		require.JSONEq(t, exceeds("3"), got)
	})

	// KNOWN BUG: calculating the complexity parses the arguments with their directives,
	// so a directive of the schema runs once for the complexity, even when the operation
	// is then rejected, and once more when the field resolves.
	// Expected: the directive runs once, when the field resolves, because argument
	// directives belong to executing the field and may check permissions or have other
	// side effects.
	// Issue: https://github.com/99designs/gqlgen/issues/4461
	t.Run("argument directives run when complexity parses arguments", func(t *testing.T) {
		got, p := complexity(t, `{ wrongTypeArg(value: "v") }`)
		require.JSONEq(t, exceeds("1000"), got)
		require.Equal(t, 1, p.directiveCalls)
		require.Equal(t, 1, p.wrongTypeArgCalls)
	})

	t.Run("argument directives run twice when the operation is accepted", func(t *testing.T) {
		resolvers := &Stub{}
		resolvers.QueryResolver.WrongTypeArg = func(ctx context.Context, value *string) (*string, error) {
			return value, nil
		}
		srv, p := newComplexityServer(resolvers, 10000)
		got := post(t, srv, `{ wrongTypeArg(value: "v") }`)
		require.JSONEq(t, `{"data":{"wrongTypeArg":"v"}}`, got)
		require.Equal(t, 2, p.directiveCalls)
	})
}
