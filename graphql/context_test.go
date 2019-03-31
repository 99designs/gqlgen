package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
)

func TestRequestContext_GetErrors(t *testing.T) {
	c := &RequestContext{
		ErrorPresenter: DefaultErrorPresenter,
	}

	ctx := context.Background()

	root := &ResolverContext{
		Field: CollectedField{
			Field: &ast.Field{
				Alias: "foo",
			},
		},
	}
	ctx = WithResolverContext(ctx, root)
	c.Error(ctx, errors.New("foo1"))
	c.Error(ctx, errors.New("foo2"))

	index := 1
	child := &ResolverContext{
		Parent: root,
		Index:  &index,
	}
	userProvidedPath := &ResolverContext{
		Parent: child,
		Field: CollectedField{
			Field: &ast.Field{
				Alias: "works",
			},
		},
	}

	ctx = WithResolverContext(ctx, child)
	c.Error(ctx, errors.New("bar"))
	c.Error(ctx, &gqlerror.Error{
		Message: "foo3",
		Path:    append(child.Path(), "works"),
	})

	specs := []struct {
		Name     string
		RCtx     *ResolverContext
		Messages []string
	}{
		{
			Name:     "with root ResolverContext",
			RCtx:     root,
			Messages: []string{"foo1", "foo2"},
		},
		{
			Name:     "with child ResolverContext",
			RCtx:     child,
			Messages: []string{"bar"},
		},
		{
			Name:     "with user provided path",
			RCtx:     userProvidedPath,
			Messages: []string{"foo3"},
		},
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			errList := c.GetErrors(spec.RCtx)
			if assert.Equal(t, len(spec.Messages), len(errList)) {
				for idx, err := range errList {
					assert.Equal(t, spec.Messages[idx], err.Message)
				}
			}
		})
	}
}

func TestGetRequestContext(t *testing.T) {
	require.Nil(t, GetRequestContext(context.Background()))

	rc := &RequestContext{}
	require.Equal(t, rc, GetRequestContext(WithRequestContext(context.Background(), rc)))
}

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
