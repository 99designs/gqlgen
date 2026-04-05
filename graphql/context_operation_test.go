package graphql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// implement context.Context interface
type testGraphRequestContext struct {
	opContext *OperationContext
}

func (t *testGraphRequestContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (t *testGraphRequestContext) Done() <-chan struct{} {
	return nil
}

func (t *testGraphRequestContext) Err() error {
	return nil
}

func (t *testGraphRequestContext) Value(key any) any {
	return t.opContext
}

func TestGetOperationContext(t *testing.T) {
	opCtx := &OperationContext{}

	t.Run("with operation context", func(t *testing.T) {
		ctx := WithOperationContext(context.Background(), opCtx)

		require.True(t, HasOperationContext(ctx))
		require.Equal(t, opCtx, GetOperationContext(ctx))
	})

	t.Run("without operation context", func(t *testing.T) {
		ctx := context.Background()

		require.False(t, HasOperationContext(ctx))
		require.Panics(t, func() {
			GetOperationContext(ctx)
		})
	})

	t.Run("with nil operation context", func(t *testing.T) {
		ctx := &testGraphRequestContext{opContext: nil}

		require.False(t, HasOperationContext(ctx))
		require.Panics(t, func() {
			GetOperationContext(ctx)
		})
	})
}

func TestCollectFields(t *testing.T) {
	getNames := func(collected []CollectedField) []string {
		names := make([]string, 0, len(collected))
		for _, f := range collected {
			names = append(names, f.Name)
		}
		return names
	}

	var (
		trueVal  = &ast.Value{Kind: ast.BooleanValue, Raw: "true"}
		falseVal = &ast.Value{Kind: ast.BooleanValue, Raw: "false"}
		skipTrue = &ast.Directive{
			Name:      "skip",
			Arguments: ast.ArgumentList{{Name: "if", Value: trueVal}},
		}
		skipFalse = &ast.Directive{
			Name:      "skip",
			Arguments: ast.ArgumentList{{Name: "if", Value: falseVal}},
		}
		includeTrue = &ast.Directive{
			Name:      "include",
			Arguments: ast.ArgumentList{{Name: "if", Value: trueVal}},
		}
		includeFalse = &ast.Directive{
			Name:      "include",
			Arguments: ast.ArgumentList{{Name: "if", Value: falseVal}},
		}
	)

	t.Run("handles fields", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "field",
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(GetOperationContext(ctx), resCtx.Field.Selections, nil)
		require.Equal(t, []string{"field"}, getNames(collected))
	})

	t.Run("handles include and skip on fields", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "fieldA",
			},
			&ast.Field{
				Name:       "fieldB",
				Directives: ast.DirectiveList{includeTrue},
			},
			&ast.Field{
				Name:       "fieldC",
				Directives: ast.DirectiveList{includeFalse},
			},
			&ast.Field{
				Name:       "fieldD",
				Directives: ast.DirectiveList{skipTrue},
			},
			&ast.Field{
				Name:       "fieldE",
				Directives: ast.DirectiveList{skipFalse},
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(GetOperationContext(ctx), resCtx.Field.Selections, nil)
		require.Equal(t, []string{"fieldA", "fieldB", "fieldE"}, getNames(collected))
	})

	t.Run("handles inline fragments that apply", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
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
			&ast.Field{
				Name: "fieldC",
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(
			GetOperationContext(ctx),
			resCtx.Field.Selections,
			[]string{"ExampleTypeB"},
		)
		require.Equal(t, []string{"fieldB", "fieldC"}, getNames(collected))
	})

	t.Run("handles inline fragment when no type", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.InlineFragment{
				TypeCondition: "",
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
			&ast.Field{
				Name: "fieldC",
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(
			GetOperationContext(ctx),
			resCtx.Field.Selections,
			[]string{"ExampleTypeB"},
		)
		require.Equal(t, []string{"fieldA", "fieldB", "fieldC"}, getNames(collected))
	})

	t.Run("handles inline fragments with include and skip", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.InlineFragment{
				TypeCondition: "",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldA*",
					},
				},
				Directives: ast.DirectiveList{includeFalse},
			},
			&ast.InlineFragment{
				TypeCondition: "",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldA1",
					},
					&ast.Field{
						Name:       "fieldA2",
						Directives: ast.DirectiveList{skipTrue},
					},
				},
				Directives: ast.DirectiveList{includeTrue},
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeB",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldB*",
					},
				},
				Directives: ast.DirectiveList{skipTrue},
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeB",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldB1",
					},
					&ast.Field{
						Name:       "fieldB2",
						Directives: ast.DirectiveList{skipTrue},
					},
				},
				Directives: ast.DirectiveList{skipFalse},
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeC",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name:       "fieldC1",
						Directives: ast.DirectiveList{includeTrue},
					},
					&ast.Field{
						Name:       "fieldC2",
						Directives: ast.DirectiveList{includeFalse},
					},
				},
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeD",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "fieldD*",
					},
				},
				Directives: ast.DirectiveList{includeTrue},
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(
			GetOperationContext(ctx),
			resCtx.Field.Selections,
			[]string{"ExampleTypeB", "ExampleTypeC"},
		)
		require.Equal(t, []string{"fieldA1", "fieldB1", "fieldC1"}, getNames(collected))
	})

	t.Run("collect inline fragments with same field name on different types", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeA",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name:             "fieldA",
						ObjectDefinition: &ast.Definition{Name: "ExampleTypeA"},
					},
				},
			},
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeB",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name:             "fieldA",
						ObjectDefinition: &ast.Definition{Name: "ExampleTypeB"},
					},
				},
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(
			GetOperationContext(ctx),
			resCtx.Field.Selections,
			[]string{"ExampleTypeA", "ExampleTypeB"},
		)
		require.Len(t, collected, 2)
		require.NotEqual(t, collected[0], collected[1])
		require.Equal(t, collected[0].Name, collected[1].Name)
	})

	t.Run("collect inline fragments with same field name and different alias", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.InlineFragment{
				TypeCondition: "ExampleTypeA",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name:             "fieldA",
						Alias:            "fieldA",
						ObjectDefinition: &ast.Definition{Name: "ExampleTypeA"},
					},
					&ast.Field{
						Name:             "fieldA",
						Alias:            "fieldA Alias",
						ObjectDefinition: &ast.Definition{Name: "ExampleTypeA"},
					},
				},
				ObjectDefinition: &ast.Definition{Name: "ExampleType", Kind: ast.Interface},
			},
		})
		resCtx := GetFieldContext(ctx)
		collected := CollectFields(
			GetOperationContext(ctx),
			resCtx.Field.Selections,
			[]string{"ExampleTypeA", "ExampleTypeB"},
		)
		require.Len(t, collected, 2)
		require.NotEqual(t, collected[0], collected[1])
		require.Equal(t, collected[0].Name, collected[1].Name)
		require.NotEqual(t, collected[0].Alias, collected[1].Alias)
	})

	t.Run("handles fragment spreads", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.FragmentSpread{
				Name: "FragmentA",
			},
			&ast.FragmentSpread{
				Name: "FragmentB",
			},
			&ast.Field{
				Name: "fieldC",
			},
		})
		resCtx := GetFieldContext(ctx)
		reqCtx := GetOperationContext(ctx)
		reqCtx.Doc = &ast.QueryDocument{
			Fragments: []*ast.FragmentDefinition{
				{
					Name:          "FragmentA",
					TypeCondition: "ExampleTypeA",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldA",
						},
					},
				},
				{
					Name:          "FragmentB",
					TypeCondition: "ExampleTypeB",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldB",
						},
					},
				},
			},
		}
		collected := CollectFields(reqCtx, resCtx.Field.Selections, []string{"ExampleTypeB"})
		require.Equal(t, []string{"fieldB", "fieldC"}, getNames(collected))
	})

	t.Run("handles fragment spreads with directives", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.FragmentSpread{
				Name:       "FragmentA",
				Directives: ast.DirectiveList{includeTrue},
			},
			&ast.FragmentSpread{
				Name:       "FragmentB",
				Directives: ast.DirectiveList{includeFalse},
			},
			&ast.FragmentSpread{
				Name:       "FragmentC",
				Directives: ast.DirectiveList{skipTrue},
			},
			&ast.FragmentSpread{
				Name:       "FragmentD",
				Directives: ast.DirectiveList{skipFalse},
			},
		})
		resCtx := GetFieldContext(ctx)
		reqCtx := GetOperationContext(ctx)
		reqCtx.Doc = &ast.QueryDocument{
			Fragments: []*ast.FragmentDefinition{
				{
					Name:          "FragmentA",
					TypeCondition: "ExampleTypeA",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldA",
						},
					},
				},
				{
					Name:          "FragmentB",
					TypeCondition: "ExampleTypeB",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldB",
						},
					},
				},
				{
					Name:          "FragmentC",
					TypeCondition: "ExampleTypeA",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldC",
						},
					},
				},
				{
					Name:          "FragmentD",
					TypeCondition: "ExampleTypeB",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldD",
						},
					},
				},
			},
		}
		collected := CollectFields(
			reqCtx,
			resCtx.Field.Selections,
			[]string{"ExampleTypeA", "ExampleTypeB"},
		)
		require.Equal(t, []string{"fieldA", "fieldD"}, getNames(collected))
	})
}

func TestCollectAllFields(t *testing.T) {
	t.Run(
		"collects all fields incl inline fragments and fragment spreads regardless of type",
		func(t *testing.T) {
			ctx := testContext(ast.SelectionSet{
				&ast.Field{
					Name: "fieldA",
				},
				&ast.InlineFragment{
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldB",
						},
					},
					Directives: ast.DirectiveList{
						&ast.Directive{Name: "someDirective"},
					},
				},
				&ast.InlineFragment{
					TypeCondition: "ExampleTypeC",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "fieldC",
						},
					},
					ObjectDefinition: &ast.Definition{Name: "ExampleTypeC"},
				},
				&ast.FragmentSpread{
					Name: "FragmentD",
				},
			})
			reqCtx := GetOperationContext(ctx)
			reqCtx.Doc = &ast.QueryDocument{
				Fragments: []*ast.FragmentDefinition{
					{
						Name:          "FragmentD",
						TypeCondition: "ExampleTypeD",
						SelectionSet: ast.SelectionSet{
							&ast.Field{
								Name: "fieldD",
							},
						},
					},
				},
			}
			ctx = WithOperationContext(ctx, reqCtx)
			require.Equal(
				t,
				[]string{"fieldA", "fieldB", "fieldC", "fieldD"},
				CollectAllFields(ctx),
			)
		},
	)

	t.Run("de-dupes aliased field names", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "field",
			},
			&ast.Field{
				Name:  "field",
				Alias: "field alias",
			},
		})
		require.Equal(t, []string{"field"}, CollectAllFields(ctx))
	})
}

func TestFieldRequested(t *testing.T) {
	t.Run("returns true when field is directly requested", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "reviews"},
			&ast.Field{Name: "title"},
		})
		require.True(t, FieldRequested(ctx, "reviews"))
		require.True(t, FieldRequested(ctx, "title"))
	})

	t.Run("returns false when field is not requested", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "title"},
		})
		require.False(t, FieldRequested(ctx, "reviews"))
	})

	t.Run("returns true when field is inside an inline fragment", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "title"},
			&ast.InlineFragment{
				TypeCondition: "Post",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "reviews"},
				},
			},
		})
		require.True(t, FieldRequested(ctx, "reviews"))
	})

	t.Run("returns true when field is inside a fragment spread", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.FragmentSpread{Name: "PostFields"},
		})
		reqCtx := GetOperationContext(ctx)
		reqCtx.Doc = &ast.QueryDocument{
			Fragments: []*ast.FragmentDefinition{
				{
					Name:          "PostFields",
					TypeCondition: "Post",
					SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "reviews"},
					},
				},
			},
		}
		ctx = WithOperationContext(ctx, reqCtx)
		require.True(t, FieldRequested(ctx, "reviews"))
	})

	t.Run("respects skip directive", func(t *testing.T) {
		trueVal := &ast.Value{Raw: "true", Kind: ast.BooleanValue}
		skipTrue := &ast.Directive{
			Name:      "skip",
			Arguments: ast.ArgumentList{{Name: "if", Value: trueVal}},
		}
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "title"},
			&ast.Field{Name: "reviews", Directives: ast.DirectiveList{skipTrue}},
		})
		require.False(t, FieldRequested(ctx, "reviews"))
	})

	t.Run("returns true for nested dot-notation path", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "reviews",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "author"},
				},
			},
		})
		require.True(t, FieldRequested(ctx, "reviews.author"))
	})

	t.Run("returns false when nested field is not present", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "reviews",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "title"},
				},
			},
		})
		require.False(t, FieldRequested(ctx, "reviews.author"))
	})

	t.Run("returns true for deeply nested dot-notation path", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "reviews",
				SelectionSet: ast.SelectionSet{
					&ast.Field{
						Name: "author",
						SelectionSet: ast.SelectionSet{
							&ast.Field{Name: "name"},
						},
					},
				},
			},
		})
		require.True(t, FieldRequested(ctx, "reviews.author.name"))
		require.False(t, FieldRequested(ctx, "reviews.author.email"))
	})

	t.Run("returns true for nested field inside an inline fragment", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "reviews",
				SelectionSet: ast.SelectionSet{
					&ast.InlineFragment{
						TypeCondition: "Review",
						SelectionSet: ast.SelectionSet{
							&ast.Field{Name: "author"},
						},
					},
				},
			},
		})
		require.True(t, FieldRequested(ctx, "reviews.author"))
	})

	t.Run("works through federation-style entity inline fragments", func(t *testing.T) {
		// In federation, the _entities field selection set contains inline fragments
		// like: ... on User { id reviews { body } } ... on Product { upc name }
		ctx := testContext(ast.SelectionSet{
			&ast.InlineFragment{
				TypeCondition: "User",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "id"},
					&ast.Field{
						Name: "reviews",
						SelectionSet: ast.SelectionSet{
							&ast.Field{Name: "body"},
						},
					},
				},
			},
			&ast.InlineFragment{
				TypeCondition: "Product",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "upc"},
					&ast.Field{Name: "name"},
				},
			},
		})
		require.True(t, FieldRequested(ctx, "id"))
		require.True(t, FieldRequested(ctx, "reviews"))
		require.True(t, FieldRequested(ctx, "reviews.body"))
		require.True(t, FieldRequested(ctx, "upc"))
		require.False(t, FieldRequested(ctx, "nonexistent"))
	})
}

func TestAnyFieldRequested(t *testing.T) {
	t.Run("returns true when any field matches", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "reviews"},
			&ast.Field{Name: "title"},
		})
		require.True(t, AnyFieldRequested(ctx, "reviews", "reviewCount", "averageRating"))
	})

	t.Run("returns false when no fields match", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "title"},
		})
		require.False(t, AnyFieldRequested(ctx, "reviews", "reviewCount", "averageRating"))
	})

	t.Run("supports dot-notation paths", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{
				Name: "reviews",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "author"},
				},
			},
		})
		require.True(t, AnyFieldRequested(ctx, "reviewCount", "reviews.author"))
		require.False(t, AnyFieldRequested(ctx, "reviewCount", "reviews.body"))
	})

	t.Run("returns false with no arguments", func(t *testing.T) {
		ctx := testContext(ast.SelectionSet{
			&ast.Field{Name: "title"},
		})
		require.False(t, AnyFieldRequested(ctx))
	})
}
