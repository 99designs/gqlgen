package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestGetResolverContext(t *testing.T) {
	require.Nil(t, GetFieldContext(context.Background()))

	rc := &FieldContext{}
	require.Equal(t, rc, GetFieldContext(WithFieldContext(context.Background(), rc)))
}

func TestNewScalarFieldContext(t *testing.T) {
	field := CollectedField{
		Field: &ast.Field{Name: "name"},
	}
	wantErr := errors.New("field of type String does not have child fields")

	fc, err := NewScalarFieldContext("User", field, true, false, wantErr)
	require.NoError(t, err)
	require.Equal(t, "User", fc.Object)
	require.Equal(t, field, fc.Field)
	require.True(t, fc.IsMethod)
	require.False(t, fc.IsResolver)

	// Child callback must always return the provided error.
	childFC, childErr := fc.Child(context.Background(), CollectedField{})
	require.Nil(t, childFC)
	require.Equal(t, wantErr, childErr)
}

func testContext(sel ast.SelectionSet) context.Context {
	ctx := context.Background()

	rqCtx := &OperationContext{}
	ctx = WithOperationContext(ctx, rqCtx)

	root := &FieldContext{
		Field: CollectedField{
			Selections: sel,
		},
	}
	ctx = WithFieldContext(ctx, root)

	return ctx
}
