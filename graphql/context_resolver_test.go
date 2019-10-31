package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/ast"
)

func TestGetResolverContext(t *testing.T) {
	require.Nil(t, GetResolverContext(context.Background()))

	rc := &ResolverContext{}
	require.Equal(t, rc, GetResolverContext(WithResolverContext(context.Background(), rc)))
}

func testContext(sel ast.SelectionSet) context.Context {

	ctx := context.Background()

	rqCtx := &RequestContext{}
	ctx = WithRequestContext(ctx, rqCtx)

	root := &ResolverContext{
		Field: CollectedField{
			Selections: sel,
		},
	}
	ctx = WithResolverContext(ctx, root)

	return ctx
}
