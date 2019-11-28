package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/ast"
)

func TestGetOperationContext(t *testing.T) {
	rc := &OperationContext{}
	require.Equal(t, rc, GetOperationContext(WithOperationContext(context.Background(), rc)))
}

func TestCollectAllFields(t *testing.T) {
	t.Run("collect fields", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "field",
			},
		})
		s := CollectAllFields(ctx)
		require.Equal(t, []string{"field"}, s)
	})

	t.Run("unique field names", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "field",
			},
			&ast.Field{
				Name:  "field",
				Alias: "field alias",
			},
		})
		s := CollectAllFields(ctx)
		require.Equal(t, []string{"field"}, s)
	})

	t.Run("collect fragments", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "fieldA",
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeA",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldA",
					},
				},
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeB",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldB",
					},
				},
			},
		})
		s := CollectAllFields(ctx)
		require.Equal(t, []string{"fieldA", "fieldB"}, s)
	})
}
