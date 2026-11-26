package graphql

import (
	"context"
	"errors"
	"fmt"
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

	fc, err := NewScalarFieldContext("User", field, true, false, "String")
	require.NoError(t, err)
	require.Equal(t, "User", fc.Object)
	require.Equal(t, field, fc.Field)
	require.True(t, fc.IsMethod)
	require.False(t, fc.IsResolver)

	// Child callback must always return an error naming the given type.
	childFC, childErr := fc.Child(context.Background(), CollectedField{})
	require.Nil(t, childFC)
	require.EqualError(t, childErr, "field of type String does not have child fields")
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

func TestNewFieldContext(t *testing.T) {
	field := CollectedField{Field: &ast.Field{Name: "name"}}
	child := NoChildFields("Leaf")

	constructors := map[string]func(
		CollectedField, string, func(context.Context, CollectedField) (*FieldContext, error), bool, bool,
	) (*FieldContext, error){
		"NewFieldContext":      NewFieldContext,
		"NewChildFieldContext": NewChildFieldContext,
	}
	for name, construct := range constructors {
		t.Run(name, func(t *testing.T) {
			fc, err := construct(field, "User", child, true, false)
			require.NoError(t, err)
			require.Equal(t, "User", fc.Object)
			require.Equal(t, field, fc.Field)
			require.True(t, fc.IsMethod)
			require.False(t, fc.IsResolver)
			require.Nil(t, fc.Args)

			// the child passed in is the one installed
			_, err = fc.Child(context.Background(), CollectedField{})
			require.EqualError(t, err, "field of type Leaf does not have child fields")
		})
	}
}

func TestSharedChildFields(t *testing.T) {
	fc, err := NoChildFields("String")(context.Background(), CollectedField{})
	require.Nil(t, fc)
	require.EqualError(t, err, "field of type String does not have child fields")

	fc, err = AbstractChildFields("INTERFACE")(context.Background(), CollectedField{})
	require.Nil(t, fc)
	require.EqualError(t, err, "FieldContext.Child cannot be called on type INTERFACE")
}

// fieldContextArgsEC stands in for a generated execution context.
type fieldContextArgsEC struct {
	err   error
	panic any
	seen  *FieldContext
}

func (ec *fieldContextArgsEC) args(
	ctx context.Context,
	rawArgs map[string]any,
) (map[string]any, error) {
	ec.seen = GetFieldContext(ctx)
	if ec.panic != nil {
		panic(ec.panic)
	}
	if ec.err != nil {
		return nil, ec.err
	}
	return map[string]any{"id": rawArgs["id"]}, nil
}

func TestNewFieldContextWithArgs(t *testing.T) {
	field := CollectedField{Field: &ast.Field{
		Name:  "user",
		Alias: "user",
		Arguments: ast.ArgumentList{
			{Name: "id", Value: &ast.Value{Kind: ast.Variable, Raw: "userID"}},
		},
		Definition: &ast.FieldDefinition{
			Name:      "user",
			Arguments: ast.ArgumentDefinitionList{{Name: "id", Type: ast.NamedType("Int", nil)}},
		},
	}}
	child := NoChildFields("User")
	recoverFunc := func(_ context.Context, r any) error { return fmt.Errorf("recovered: %v", r) }
	newContexts := func() (context.Context, *OperationContext) {
		ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, recoverFunc)
		return ctx, &OperationContext{
			Variables:   map[string]any{"userID": int64(7)},
			RecoverFunc: recoverFunc,
		}
	}
	build := func(
		ctx context.Context,
		oc *OperationContext,
		ec *fieldContextArgsEC,
		recoverFromPanic bool,
	) (*FieldContext, error) {
		return NewFieldContextWithArgs(
			ctx,
			oc,
			ec,
			field,
			(*fieldContextArgsEC).args,
			"Query",
			child,
			true,
			true,
			recoverFromPanic,
		)
	}

	t.Run("parses arguments with the field context in scope", func(t *testing.T) {
		ctx, oc := newContexts()
		ec := &fieldContextArgsEC{}

		fc, err := build(ctx, oc, ec, true)
		require.NoError(t, err)
		require.Equal(t, "Query", fc.Object)
		require.Equal(t, field, fc.Field)
		require.True(t, fc.IsMethod)
		require.True(t, fc.IsResolver)
		require.Equal(t, map[string]any{"id": int64(7)}, fc.Args)
		require.Same(t, fc, ec.seen)
		require.Empty(t, GetErrors(ctx))
	})

	t.Run("reports an argument error and returns it", func(t *testing.T) {
		ctx, oc := newContexts()
		ec := &fieldContextArgsEC{err: errors.New("bad id")}

		fc, err := build(ctx, oc, ec, true)
		require.EqualError(t, err, "bad id")
		require.NotNil(t, fc)
		require.Nil(t, fc.Args)
		require.Len(t, GetErrors(ctx), 1)
		require.Equal(t, "bad id", GetErrors(ctx)[0].Message)
	})

	t.Run("recovers a panic when asked to", func(t *testing.T) {
		ctx, oc := newContexts()
		ec := &fieldContextArgsEC{panic: "boom"}

		fc, err := build(ctx, oc, ec, true)
		require.ErrorContains(t, err, "recovered: boom")
		require.NotNil(t, fc)
		require.Len(t, GetErrors(ctx), 1)
		require.Equal(t, "recovered: boom", GetErrors(ctx)[0].Message)
		require.Equal(t, ast.Path{ast.PathName("user")}, GetErrors(ctx)[0].Path)
	})

	t.Run("lets a panic through otherwise", func(t *testing.T) {
		ctx, oc := newContexts()
		ec := &fieldContextArgsEC{panic: "boom"}

		require.PanicsWithValue(t, "boom", func() { _, _ = build(ctx, oc, ec, false) })
		require.Empty(t, GetErrors(ctx))
	})
}
