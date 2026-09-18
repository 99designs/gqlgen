package graphql

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type inputTestObject struct {
	Name string
}

func unmarshalInputTestObject(_ context.Context, obj any) (inputTestObject, error) {
	m, ok := obj.(map[string]any)
	if !ok {
		return inputTestObject{}, errors.New("not a map")
	}
	return inputTestObject{Name: m["name"].(string)}, nil
}

// inputTestEC stands in for the generated *executionContext that
// use_function_syntax_for_execution_context passes to unmarshalers as an argument.
type inputTestEC struct{ calls int }

func unmarshalInputTestObjectWithEC(
	ctx context.Context,
	ec *inputTestEC,
	obj any,
) (inputTestObject, error) {
	ec.calls++
	return unmarshalInputTestObject(ctx, obj)
}

func TestUnmarshalInputFromContext(t *testing.T) {
	built := BuildUnmarshalerMap(unmarshalInputTestObject)

	tests := []struct {
		name    string
		ctx     func() context.Context
		target  any
		wantErr string
		want    inputTestObject
	}{
		{
			name:    "no unmarshaler map in context",
			ctx:     context.Background,
			target:  &inputTestObject{},
			wantErr: "graphql: the input context is empty",
		},
		{
			name:    "nil build function",
			ctx:     func() context.Context { return WithLazyUnmarshalerMap(context.Background(), nil) },
			target:  &inputTestObject{},
			wantErr: "graphql: the input context is empty",
		},
		{
			name: "build function returns nil map",
			ctx: func() context.Context {
				return WithLazyUnmarshalerMap(
					context.Background(),
					func() UnmarshalerMap { return nil },
				)
			},
			target:  &inputTestObject{},
			wantErr: "graphql: the input context is empty",
		},
		{
			name:    "target is not a pointer",
			ctx:     func() context.Context { return WithUnmarshalerMap(context.Background(), built) },
			target:  inputTestObject{},
			wantErr: "graphql: input must be a non-nil pointer",
		},
		{
			name:    "target is a nil pointer",
			ctx:     func() context.Context { return WithUnmarshalerMap(context.Background(), built) },
			target:  (*inputTestObject)(nil),
			wantErr: "graphql: input must be a non-nil pointer",
		},
		{
			name:    "no unmarshaler for target type",
			ctx:     func() context.Context { return WithUnmarshalerMap(context.Background(), built) },
			target:  &struct{ Other int }{},
			wantErr: "graphql: no unmarshal function found",
		},
		{
			name:   "unmarshals into target",
			ctx:    func() context.Context { return WithUnmarshalerMap(context.Background(), built) },
			target: &inputTestObject{},
			want:   inputTestObject{Name: "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalInputFromContext(tt.ctx(), map[string]any{"name": "bob"}, tt.target)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, *tt.target.(*inputTestObject))
		})
	}
}

// UnmarshalerMap must stay an alias, not become a defined type. Passing a plain map as
// an argument would keep compiling either way, but these two spellings only stay
// interchangeable while the types are identical.
var (
	_ func(context.Context, map[reflect.Type]reflect.Value) context.Context = WithUnmarshalerMap
	_ func(...any) map[reflect.Type]reflect.Value                           = BuildUnmarshalerMap
)

func TestWithLazyUnmarshalerMapBuildsOnceOnFirstUse(t *testing.T) {
	var builds int
	ctx := WithLazyUnmarshalerMap(context.Background(), func() UnmarshalerMap {
		builds++
		return BuildUnmarshalerMap(unmarshalInputTestObject)
	})
	assert.Zero(t, builds, "map was built before it was needed")

	for range 3 {
		var got inputTestObject
		require.NoError(t, UnmarshalInputFromContext(ctx, map[string]any{"name": "bob"}, &got))
		assert.Equal(t, inputTestObject{Name: "bob"}, got)
	}
	assert.Equal(t, 1, builds)
}

// Resolvers within one request run concurrently and share its context, so the build
// function must be called exactly once no matter how many of them unmarshal input.
func TestWithLazyUnmarshalerMapConcurrent(t *testing.T) {
	const goroutines = 50

	// Safe to increment unguarded: the point of the test is that this runs once.
	var builds int
	ctx := WithLazyUnmarshalerMap(context.Background(), func() UnmarshalerMap {
		builds++
		return BuildUnmarshalerMap(unmarshalInputTestObject)
	})

	// Each goroutine owns one slot, so results need no locking, and assertions stay on
	// the test goroutine where require is allowed to call FailNow.
	got := make([]inputTestObject, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			errs[i] = UnmarshalInputFromContext(ctx, map[string]any{"name": "bob"}, &got[i])
		})
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "goroutine %d", i)
		assert.Equal(t, inputTestObject{Name: "bob"}, got[i])
	}
	assert.Equal(t, 1, builds, "build function must run exactly once per context")
}

// BindUnmarshaler exists so that schemas generated with
// use_function_syntax_for_execution_context, whose unmarshalers take the execution
// context as an argument, still produce entries BuildUnmarshalerMap will store.
func TestBindUnmarshaler(t *testing.T) {
	ec := &inputTestEC{}
	ctx := WithUnmarshalerMap(
		context.Background(),
		BuildUnmarshalerMap(BindUnmarshaler(ec, unmarshalInputTestObjectWithEC)),
	)

	var got inputTestObject
	require.NoError(t, UnmarshalInputFromContext(ctx, map[string]any{"name": "bob"}, &got))
	assert.Equal(t, inputTestObject{Name: "bob"}, got)
	assert.Equal(t, 1, ec.calls, "bound execution context was not passed through")
}

// An unmarshaler that still takes its execution context explicitly cannot be invoked
// with (ctx, raw), so storing it would panic inside reflect at call time.
func TestBuildUnmarshalerMapSkipsWrongShapes(t *testing.T) {
	maps := BuildUnmarshalerMap(
		unmarshalInputTestObjectWithEC, // three arguments
		"not a function",
		func(context.Context, any) {}, // no return values
		unmarshalInputTestObject,      // the only usable entry
	)

	assert.Len(t, maps, 1)
	assert.Contains(t, maps, reflect.TypeFor[inputTestObject]())
}
