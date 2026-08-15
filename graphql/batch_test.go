package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestBatchErrorList_UnwrapFiltersNil(t *testing.T) {
	sentinel := errors.New("sentinel")
	list := BatchErrorList{nil, sentinel, nil}

	type unwrapper interface {
		Unwrap() []error
	}
	u, ok := any(list).(unwrapper)
	require.True(t, ok)

	got := u.Unwrap()
	require.Len(t, got, 1)
	require.Equal(t, sentinel, got[0])
}

func TestBatchErrorList_ErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	other := errors.New("other")
	list := BatchErrorList{nil, sentinel, other}

	require.ErrorIs(t, list, sentinel)
	require.ErrorIs(t, list, other)
	require.NotErrorIs(t, list, errors.New("missing"))
}

func TestBatchErrorList_ErrorsIsWithAllNil(t *testing.T) {
	list := BatchErrorList{nil, nil}

	require.NotErrorIs(t, list, errors.New("missing"))
}

func TestWithBatchParentValues(t *testing.T) {
	type edge struct{ ID string }

	cases := map[string]struct {
		values []edge
	}{
		"nil":      {values: nil},
		"empty":    {values: []edge{}},
		"single":   {values: []edge{{ID: "a"}}},
		"multiple": {values: []edge{{ID: "a"}, {ID: "b"}, {ID: "c"}}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := WithBatchParentValues(context.Background(), "Edge", tc.values)

			group := GetBatchParentGroup(ctx, "Edge")
			require.NotNil(t, group)
			require.Nil(t, group.IndexMap)

			// Generated batch resolvers recover the parents with exactly this
			// assertion; a []edge here is the bug this helper exists to prevent.
			parents, ok := group.Parents.([]*edge)
			require.True(t, ok)
			require.Len(t, parents, len(tc.values))

			// Parents must alias the source slice rather than copy it, so that
			// parents[i] is the same object the marshaler reports as fc.Result.
			for i := range tc.values {
				require.Same(t, &tc.values[i], parents[i])
			}
		})
	}
}

func newBatchTestContext() context.Context {
	ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, nil)
	ctx = WithPathContext(ctx, NewPathWithField("users"))
	ctx = WithPathContext(ctx, NewPathWithIndex(0))
	ctx = WithPathContext(ctx, NewPathWithField("profile"))
	return ctx
}

func TestResolveBatchGroupResult_Success(t *testing.T) {
	ctx := newBatchTestContext()
	result := &BatchFieldResult{
		Results: []string{"a", "b"},
	}

	got, err := ResolveBatchGroupResult[string](
		ctx,
		ast.PathIndex(1),
		2,
		result,
		"User.profile",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, "b", got)
	require.Empty(t, GetErrors(ctx))
}

func TestResolveBatchGroupResult_ResultLenMismatch(t *testing.T) {
	ctx := newBatchTestContext()
	result := &BatchFieldResult{
		Results: []string{"a"},
	}

	got, err := ResolveBatchGroupResult[string](
		ctx,
		ast.PathIndex(1),
		2,
		result,
		"User.profile",
		nil,
	)
	require.NoError(t, err)
	require.Nil(t, got)

	errs := GetErrors(ctx)
	require.Len(t, errs, 1)
	require.Equal(
		t,
		"index 1: batch resolver User.profile returned 1 results for 2 "+
			"parents",
		errs[0].Message,
	)
	require.Equal(
		t,
		ast.Path{
			ast.PathName("users"),
			ast.PathIndex(1),
			ast.PathName("profile"),
		},
		errs[0].Path,
	)
}

func TestResolveBatchSingleResult_BatchErrors(t *testing.T) {
	ctx := newBatchTestContext()

	got, err := ResolveBatchSingleResult[string](
		ctx,
		[]string{"a"},
		BatchErrorList{errors.New("boom")},
		"User.profile",
	)
	require.NoError(t, err)
	require.Nil(t, got)

	errs := GetErrors(ctx)
	require.Len(t, errs, 1)
	require.Equal(t, "boom", errs[0].Message)
}

func TestResolveBatchSingleResult_ErrorLenMismatch(t *testing.T) {
	ctx := newBatchTestContext()

	got, err := ResolveBatchSingleResult[string](
		ctx,
		[]string{"a"},
		BatchErrorList{},
		"User.profile",
	)
	require.NoError(t, err)
	require.Nil(t, got)

	errs := GetErrors(ctx)
	require.Len(t, errs, 1)
	require.Equal(
		t,
		"batch resolver User.profile returned 0 errors for 1 "+
			"parents (index 0)",
		errs[0].Message,
	)
}

func TestBatchParentsFor(t *testing.T) {
	type edge struct{ ID string }
	edges := []edge{{ID: "a"}, {ID: "b"}}

	// withIndex puts a parent index on the path, as the slice marshaler does.
	withIndex := func(ctx context.Context) context.Context {
		ctx = WithPathContext(ctx, NewPathWithField("edges"))
		ctx = WithPathContext(ctx, NewPathWithIndex(1))
		return WithPathContext(ctx, NewPathWithField("node"))
	}

	cases := map[string]struct {
		ctx     func() context.Context
		wantOK  bool
		wantIdx ast.PathIndex
	}{
		"batched": {
			ctx: func() context.Context {
				return withIndex(WithBatchParentValues(context.Background(), "Edge", edges))
			},
			wantOK:  true,
			wantIdx: 1,
		},
		"no group registered": {
			ctx:    func() context.Context { return withIndex(context.Background()) },
			wantOK: false,
		},
		"group registered for another type": {
			ctx: func() context.Context {
				return withIndex(WithBatchParentValues(context.Background(), "Other", edges))
			},
			wantOK: false,
		},
		"group holds a different Go type": {
			ctx: func() context.Context {
				return withIndex(WithBatchParents(context.Background(), "Edge", edges, nil))
			},
			wantOK: false,
		},
		"no parent index on the path": {
			ctx: func() context.Context {
				return WithBatchParentValues(context.Background(), "Edge", edges)
			},
			wantOK: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			batch, ok := BatchParentsFor[*edge](tc.ctx(), "Edge")
			require.Equal(t, tc.wantOK, ok)
			if !tc.wantOK {
				// Callers fall back to a single-parent call; nothing else may be read.
				require.Empty(t, batch.Parents)
				return
			}
			require.Equal(t, tc.wantIdx, batch.Index)
			require.Len(t, batch.Parents, len(edges))
			require.Same(t, &edges[0], batch.Parents[0])
		})
	}
}

func TestBatchParents_FieldResultMemoizesPerResponseKey(t *testing.T) {
	type edge struct{ ID string }
	edges := []edge{{ID: "a"}, {ID: "b"}}
	ctx := WithBatchParentValues(context.Background(), "Edge", edges)
	ctx = WithPathContext(ctx, NewPathWithField("edges"))
	ctx = WithPathContext(ctx, NewPathWithIndex(0))
	ctx = WithPathContext(ctx, NewPathWithField("node"))

	batch, ok := BatchParentsFor[*edge](ctx, "Edge")
	require.True(t, ok)

	var calls int
	resolve := func() (any, error) {
		calls++
		return "resolved", nil
	}

	// Both sibling parents resolve the same field: one call, shared result.
	node := CollectedField{Field: &ast.Field{Name: "node", Alias: "node"}}
	require.Equal(t, "resolved", batch.FieldResult(node, resolve).Results)
	require.Equal(t, "resolved", batch.FieldResult(node, resolve).Results)
	require.Equal(t, 1, calls)

	// A different alias is a different response key, so it resolves separately.
	aliased := CollectedField{Field: &ast.Field{Name: "node", Alias: "other"}}
	require.Equal(t, "resolved", batch.FieldResult(aliased, resolve).Results)
	require.Equal(t, 2, calls)

	// An empty alias falls back to the field name, sharing the first key.
	unaliased := CollectedField{Field: &ast.Field{Name: "node"}}
	require.Equal(t, "resolved", batch.FieldResult(unaliased, resolve).Results)
	require.Equal(t, 2, calls)
}
