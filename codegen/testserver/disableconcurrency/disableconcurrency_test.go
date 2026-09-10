//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package disableconcurrency

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func newServer() *client.Client {
	resolvers := &Stub{}
	resolvers.QueryResolver.Methods = func(ctx context.Context) (*Methods, error) {
		return &Methods{}, nil
	}
	resolvers.QueryResolver.InlineObject = func(ctx context.Context) (*InlineObject, error) {
		return &InlineObject{}, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	return client.New(srv)
}

// @disableConcurrency changes only how a field is dispatched, not its result.
func TestDisableConcurrency_Runtime(t *testing.T) {
	c := newServer()

	var resp struct {
		Methods struct {
			WithContext       bool
			WithContextInline bool
			NoContext         bool
		}
		InlineObject struct {
			A bool
			B bool
		}
	}

	err := c.Post(`query {
		methods { withContext withContextInline noContext }
		inlineObject { a b }
	}`, &resp)
	require.NoError(t, err)

	require.True(t, resp.Methods.WithContext)
	require.True(t, resp.Methods.WithContextInline)
	require.True(t, resp.Methods.NoContext)
	require.True(t, resp.InlineObject.A)
	require.True(t, resp.InlineObject.B)
}

// Assert the emitted dispatch code: @disableConcurrency fields take the inline
// path while an unannotated context-taking field still spawns a goroutine.
func TestDisableConcurrency_GeneratedDispatch(t *testing.T) {
	src, err := os.ReadFile("generated.go")
	require.NoError(t, err)
	gen := string(src)

	// Unannotated context-taking method stays concurrent: it never takes the
	// inline path and its resolver runs inside the concurrent innerFunc.
	require.NotContains(t, gen, "out.Values[i] = ec._Methods_withContext(ctx, field, obj)")
	require.Contains(t, gen, "res = ec._Methods_withContext(ctx, field, obj)")

	// @disableConcurrency on a context-taking method: resolved inline.
	require.Contains(t, gen, "out.Values[i] = ec._Methods_withContextInline(ctx, field, obj)")
	require.NotContains(t, gen, "res = ec._Methods_withContextInline(ctx, field, obj)")

	// Object-level @disableConcurrency: every context-taking field resolved inline.
	require.Contains(t, gen, "out.Values[i] = ec._InlineObject_a(ctx, field, obj)")
	require.Contains(t, gen, "out.Values[i] = ec._InlineObject_b(ctx, field, obj)")
	require.NotContains(t, gen, "res = ec._InlineObject_a(ctx, field, obj)")
}
