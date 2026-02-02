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
