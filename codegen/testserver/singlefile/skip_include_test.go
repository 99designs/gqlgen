package singlefile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestSkipInclude(t *testing.T) {
	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	a := func() *string { a := "a"; return &a }()
	b := func() *string { b := "b"; return &b }()
	resolvers.QueryResolver.SkipInclude = func(ctx context.Context) (*SkipIncludeTestType, error) {
		return &SkipIncludeTestType{A: a, B: b}, nil
	}

	// Taken verbatim from the test cases found at the reference graphql-js implementation at:
	// https://github.com/graphql/graphql-js/blob/2120ff3f08a0e379e41a33f3c1a8c6127e0e574c/src/execution/__tests__/directives-test.ts
	// last updated on 2022-03-28 as of 2025-05-19.

	t.Run("works without directives", func(t *testing.T) {
		var r struct{ SkipInclude *SkipIncludeTestType }
		c.MustPost(`query { skipInclude { a, b } }`, &r)
		assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
	})
	t.Run("works on scalars", func(t *testing.T) {
		t.Run("if true includes scalar", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude { a, b @include(if: true) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})
		t.Run("if false omits on scalar", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude{ a, b @include(if: false) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})
		t.Run("unless false includes scalar", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude{ a, b @skip(if: false) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})
		t.Run("unless true omits scalar", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude{ a, b @skip(if: true) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})
	})
	t.Run("works on fragment spreads", func(t *testing.T) {
		t.Run("if false omits fragment spread", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ...Frag @include(if: false) } } fragment Frag on SkipIncludeTestType { b }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})

		t.Run("if true includes fragment spread", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ...Frag @include(if: true) } } fragment Frag on SkipIncludeTestType { b }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("unless false includes fragment spread", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ...Frag @skip(if: false) } } fragment Frag on SkipIncludeTestType { b }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("unless true omits fragment spread", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ...Frag @skip(if: true) } } fragment Frag on SkipIncludeTestType { b }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})
	})
	t.Run("works on inline fragment", func(t *testing.T) {
		t.Run("if false omits inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ... on SkipIncludeTestType @include(if: false) { b } } }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})

		t.Run("if true includes inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ... on SkipIncludeTestType @include(if: true) { b } } }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("unless false includes inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ... on SkipIncludeTestType @skip(if: false) { b } } }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("unless true includes inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(
				`query { skipInclude { a ... on SkipIncludeTestType @skip(if: true) { b } } }`,
				&r,
			)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})
	})
	t.Run("works on anonymous inline fragment", func(t *testing.T) {
		t.Run("if false omits anonymous inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`query { skipInclude { a ... @include(if: false) { b } } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})

		t.Run("if true includes anonymous inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`query { skipInclude { a ... @include(if: true) { b } } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("unless false includes anonymous inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`query Q { skipInclude { a ... @skip(if: false) { b } } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("unless true includes anonymous inline fragment", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`query { skipInclude { a ... @skip(if: true) { b } } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})
	})
	t.Run("works with skip and include directives", func(t *testing.T) {
		t.Run("include and no skip", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude{ a b @include(if: true) @skip(if: false) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{a, b}, r.SkipInclude)
		})

		t.Run("include and skip", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude{ a b @include(if: true) @skip(if: true) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})

		t.Run("no include or skip", func(t *testing.T) {
			var r struct{ SkipInclude *SkipIncludeTestType }
			c.MustPost(`{ skipInclude{ a b @include(if: false) @skip(if: false) } }`, &r)
			assert.Equal(t, &SkipIncludeTestType{A: a}, r.SkipInclude)
		})
	})
}
