package singlefile

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
)

func TestComplexityCollisions(t *testing.T) {
	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	resolvers.QueryResolver.Overlapping = func(ctx context.Context) (fields *OverlappingFields, e error) {
		return &OverlappingFields{
			Foo:    2,
			NewFoo: 3,
		}, nil
	}

	resolvers.OverlappingFieldsResolver.OldFoo = func(ctx context.Context, obj *OverlappingFields) (i int, e error) {
		return obj.Foo, nil
	}

	var resp struct {
		Overlapping struct {
			OneFoo  int `json:"oneFoo"`
			TwoFoo  int `json:"twoFoo"`
			OldFoo  int `json:"oldFoo"`
			NewFoo  int `json:"newFoo"`
			New_foo int `json:"new_foo"`
		}
	}
	c.MustPost(`query { overlapping { oneFoo, twoFoo, oldFoo, newFoo, new_foo } }`, &resp)
	require.Equal(t, 2, resp.Overlapping.OneFoo)
	require.Equal(t, 2, resp.Overlapping.TwoFoo)
	require.Equal(t, 2, resp.Overlapping.OldFoo)
	require.Equal(t, 3, resp.Overlapping.NewFoo)
	require.Equal(t, 3, resp.Overlapping.New_foo)
}

func TestComplexityFuncs(t *testing.T) {
	resolvers := &Stub{}
	cfg := Config{Resolvers: resolvers}
	cfg.Complexity.OverlappingFields.Foo = func(childComplexity int) int { return 1000 }
	cfg.Complexity.OverlappingFields.NewFoo = func(childComplexity int) int { return 5 }

	srv := handler.New(NewExecutableSchema(cfg))
	srv.AddTransport(transport.POST{})
	srv.Use(extension.FixedComplexityLimit(10))
	c := client.New(srv)

	resolvers.QueryResolver.Overlapping = func(ctx context.Context) (fields *OverlappingFields, e error) {
		return &OverlappingFields{
			Foo:    2,
			NewFoo: 3,
		}, nil
	}

	t.Run("with high complexity limit will not run", func(t *testing.T) {
		ran := false
		resolvers.OverlappingFieldsResolver.OldFoo = func(ctx context.Context, obj *OverlappingFields) (i int, e error) {
			ran = true
			return obj.Foo, nil
		}

		var resp struct {
			Overlapping any
		}
		err := c.Post(`query { overlapping { oneFoo, twoFoo, oldFoo, newFoo, new_foo } }`, &resp)

		require.EqualError(t, err, `[{"message":"operation has complexity 2012, which exceeds the limit of 10","extensions":{"code":"COMPLEXITY_LIMIT_EXCEEDED"}}]`)
		require.False(t, ran)
	})

	t.Run("with low complexity will run", func(t *testing.T) {
		ran := false
		resolvers.QueryResolver.Overlapping = func(ctx context.Context) (fields *OverlappingFields, e error) {
			ran = true
			return &OverlappingFields{
				Foo:    2,
				NewFoo: 3,
			}, nil
		}

		var resp struct {
			Overlapping any
		}
		c.MustPost(`query { overlapping { newFoo } }`, &resp)

		require.True(t, ran)
	})

	t.Run("with multiple low complexity will not run", func(t *testing.T) {
		ran := false
		resolvers.QueryResolver.Overlapping = func(ctx context.Context) (fields *OverlappingFields, e error) {
			ran = true
			return &OverlappingFields{
				Foo:    2,
				NewFoo: 3,
			}, nil
		}

		var resp any
		err := c.Post(`query {
			a: overlapping { newFoo },
			b: overlapping { newFoo },
			c: overlapping { newFoo },
		}`, &resp)

		require.EqualError(t, err, `[{"message":"operation has complexity 18, which exceeds the limit of 10","extensions":{"code":"COMPLEXITY_LIMIT_EXCEEDED"}}]`)
		require.False(t, ran)
	})
}
