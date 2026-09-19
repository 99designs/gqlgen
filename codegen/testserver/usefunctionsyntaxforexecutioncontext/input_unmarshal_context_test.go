package usefunctionsyntaxforexecutioncontext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

// use_function_syntax_for_execution_context generates unmarshalers as free functions
// taking *executionContext, which the index cannot invoke directly. Without
// graphql.BindUnmarshaler this call panicked with "reflect: Call with too few input
// arguments", which the resolver recovery turned into an opaque internal error.
func TestUnmarshalInputFromContextWithFunctionSyntax(t *testing.T) {
	var (
		got    CreateUserInput
		gotErr error
		called bool
	)

	resolvers := &Stub{}
	resolvers.QueryResolver.ListUsers = func(ctx context.Context, filter *UserFilter) ([]*User, error) {
		called = true
		gotErr = graphql.UnmarshalNamedInputFromContext(
			ctx, "CreateUserInput", map[string]any{"name": "bob"}, &got,
		)
		return nil, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Log: func(ctx context.Context, _ any, next graphql.Resolver, _ *string) (any, error) {
				return next(ctx)
			},
		},
	}))

	var resp struct {
		ListUsers []struct{ Name string }
	}
	require.NoError(t, client.New(srv).Post(`query { listUsers { name } }`, &resp))

	require.True(t, called, "resolver never ran, so nothing was exercised")
	require.NoError(t, gotErr)
	assert.Equal(t, "bob", got.Name)
}
