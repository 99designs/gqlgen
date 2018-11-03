package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/ast"
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
	ctx = WithResolverContext(ctx, child)
	c.Error(ctx, errors.New("bar"))

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
