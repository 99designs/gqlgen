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

// newUnmarshalInputFromContextClient serves a schema whose defaultScalar resolver
// unmarshals a DefaultInput through graphql.UnmarshalInputFromContext, and records what
// each call saw.
func newUnmarshalInputFromContextClient(
	t *testing.T,
) (*client.Client, func() ([]DefaultInput, []error)) {
	t.Helper()

	var (
		mu     sync.Mutex
		inputs []DefaultInput
		errs   []error
	)

	resolvers := &Stub{}
	resolvers.QueryResolver.DefaultScalar = func(ctx context.Context, arg string) (string, error) {
		var in DefaultInput
		err := graphql.UnmarshalInputFromContext(ctx, map[string]any{"falsyBoolean": true}, &in)

		mu.Lock()
		defer mu.Unlock()
		inputs = append(inputs, in)
		errs = append(errs, err)
		return arg, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	return client.New(srv), func() ([]DefaultInput, []error) {
		mu.Lock()
		defer mu.Unlock()
		return inputs, errs
	}
}

func TestUnmarshalInputFromContextInResolver(t *testing.T) {
	c, recorded := newUnmarshalInputFromContextClient(t)

	var resp struct{ DefaultScalar string }
	c.MustPost(`query { defaultScalar(arg: "hi") }`, &resp)

	inputs, errs := recorded()
	require.Len(t, errs, 1)
	require.NoError(t, errs[0])
	require.NotNil(t, inputs[0].FalsyBoolean)
	assert.True(t, *inputs[0].FalsyBoolean)
}

// The unmarshaler map is bound to the execution context of the request that built it, so
// concurrent requests must each get a working map of their own.
func TestUnmarshalInputFromContextConcurrentRequests(t *testing.T) {
	const requests = 50

	c, recorded := newUnmarshalInputFromContextClient(t)

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

	inputs, errs := recorded()
	require.Len(t, errs, requests)
	for _, err := range errs {
		require.NoError(t, err)
	}
	for _, in := range inputs {
		require.NotNil(t, in.FalsyBoolean)
		assert.True(t, *in.FalsyBoolean)
	}
}
