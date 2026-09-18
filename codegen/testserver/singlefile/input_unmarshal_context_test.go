package singlefile

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

// unmarshalFromResolver runs one query whose defaultScalar resolver calls unmarshal with
// the request's context, and reports what that call saw.
func unmarshalFromResolver(
	t *testing.T,
	unmarshal func(ctx context.Context) error,
) error {
	t.Helper()

	var (
		called bool
		got    error
	)

	resolvers := &Stub{}
	resolvers.QueryResolver.DefaultScalar = func(ctx context.Context, arg string) (string, error) {
		called = true
		got = unmarshal(ctx)
		return arg, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))

	var resp struct{ DefaultScalar string }
	require.NoError(t, client.New(srv).Post(`query { defaultScalar(arg: "hi") }`, &resp))
	require.True(t, called, "the resolver never ran, so nothing was exercised")

	return got
}

func TestUnmarshalNamedInputFromContextInResolver(t *testing.T) {
	var got DefaultInput

	err := unmarshalFromResolver(t, func(ctx context.Context) error {
		return graphql.UnmarshalNamedInputFromContext(
			ctx, "DefaultInput", map[string]any{"falsyBoolean": true}, &got,
		)
	})

	require.NoError(t, err)
	require.NotNil(t, got.FalsyBoolean)
	assert.True(t, *got.FalsyBoolean)
}

// Eight of this schema's inputs unmarshal to map[string]any, so the Go type cannot tell
// them apart and a type-keyed lookup silently returned whichever was registered last.
// Selecting by GraphQL name keeps them distinct.
func TestUnmarshalNamedInputFromContextDistinguishesMapInputs(t *testing.T) {
	var changes, searchFilters map[string]any

	err := unmarshalFromResolver(t, func(ctx context.Context) error {
		if err := graphql.UnmarshalNamedInputFromContext(
			ctx, "Changes", map[string]any{"a": 1}, &changes,
		); err != nil {
			return err
		}
		return graphql.UnmarshalNamedInputFromContext(
			ctx, "SearchFilters", map[string]any{"category": "books"}, &searchFilters,
		)
	})

	require.NoError(t, err)

	// Each input keeps its own field set, which one shared map[string]any key could not
	// represent: "a" belongs to Changes and "category" to SearchFilters. Assert on the
	// field sets rather than the values, so the test does not depend on how a nullable
	// field is represented.
	assert.Contains(t, changes, "a")
	assert.NotContains(t, changes, "category")
	assert.Contains(t, searchFilters, "category")
	assert.NotContains(t, searchFilters, "a")
}

func TestUnmarshalNamedInputFromContextUnknownInput(t *testing.T) {
	var got DefaultInput

	err := unmarshalFromResolver(t, func(ctx context.Context) error {
		return graphql.UnmarshalNamedInputFromContext(ctx, "NoSuchInput", map[string]any{}, &got)
	})

	require.ErrorContains(t, err, `no unmarshaler for input "NoSuchInput"`)
}

// The index is bound to the execution context of the request that built it, so
// concurrent requests must each get a working one of their own.
func TestUnmarshalNamedInputFromContextConcurrentRequests(t *testing.T) {
	const requests = 50

	var (
		mu     sync.Mutex
		inputs []DefaultInput
		errs   []error
	)

	resolvers := &Stub{}
	resolvers.QueryResolver.DefaultScalar = func(ctx context.Context, arg string) (string, error) {
		var in DefaultInput
		err := graphql.UnmarshalNamedInputFromContext(
			ctx, "DefaultInput", map[string]any{"falsyBoolean": true}, &in,
		)

		mu.Lock()
		defer mu.Unlock()
		inputs = append(inputs, in)
		errs = append(errs, err)
		return arg, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	c := client.New(srv)

	// Post rather than MustPost: MustPost panics, which from a spawned goroutine takes
	// down the test binary instead of failing this test.
	postErrs := make([]error, requests)

	var wg sync.WaitGroup
	for i := range requests {
		wg.Go(func() {
			var resp struct{ DefaultScalar string }
			postErrs[i] = c.Post(`query { defaultScalar(arg: "hi") }`, &resp)
		})
	}
	wg.Wait()

	for i, err := range postErrs {
		require.NoErrorf(t, err, "request %d", i)
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, errs, requests)
	for _, err := range errs {
		require.NoError(t, err)
	}
	for _, in := range inputs {
		require.NotNil(t, in.FalsyBoolean)
		assert.True(t, *in.FalsyBoolean)
	}
}
