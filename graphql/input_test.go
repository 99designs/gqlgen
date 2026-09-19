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

// Two GraphQL inputs commonly unmarshal to the same Go type: every map-backed input
// object unmarshals to map[string]any. inputTestMap stands in for that case.
type inputTestMap = map[string]any

func unmarshalInputTestObject(_ context.Context, obj any) (inputTestObject, error) {
	m, ok := obj.(map[string]any)
	if !ok {
		return inputTestObject{}, errors.New("not a map")
	}
	return inputTestObject{Name: m["name"].(string)}, nil
}

func unmarshalInputTestObjectPointer(ctx context.Context, obj any) (*inputTestObject, error) {
	out, err := unmarshalInputTestObject(ctx, obj)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// unmarshalChangesLike and unmarshalFiltersLike both unmarshal to inputTestMap, so the Go
// type cannot tell them apart and only the GraphQL name can.
func unmarshalChangesLike(_ context.Context, obj any) (inputTestMap, error) {
	return inputTestMap{"from": "changes", "raw": obj}, nil
}

func unmarshalFiltersLike(_ context.Context, obj any) (inputTestMap, error) {
	return inputTestMap{"from": "filters", "raw": obj}, nil
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

func namedIndexContext(t *testing.T) context.Context {
	t.Helper()
	return WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("TestObject", unmarshalInputTestObject),
		NewInputUnmarshaler("Changes", unmarshalChangesLike),
		NewInputUnmarshaler("Filters", unmarshalFiltersLike),
	))
}

// Selecting by GraphQL name resolves inputs that share a Go type, which is the whole
// point of indexing by name rather than by reflect.Type.
func TestUnmarshalNamedInputFromContextDistinguishesSharedGoType(t *testing.T) {
	ctx := namedIndexContext(t)

	var changes inputTestMap
	require.NoError(t, UnmarshalNamedInputFromContext(ctx, "Changes", nil, &changes))

	var filters inputTestMap
	require.NoError(t, UnmarshalNamedInputFromContext(ctx, "Filters", nil, &filters))

	assert.Equal(t, "changes", changes["from"])
	assert.Equal(t, "filters", filters["from"])
}

func TestUnmarshalNamedInputFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     func(*testing.T) context.Context
		input   string
		target  any
		wantErr string
	}{
		{
			name:    "no index in context",
			ctx:     func(*testing.T) context.Context { return context.Background() },
			input:   "TestObject",
			target:  &inputTestObject{},
			wantErr: "graphql: the input context is empty",
		},
		{
			name: "nil build function",
			ctx: func(*testing.T) context.Context {
				return WithLazyInputUnmarshalerIndex(context.Background(), nil)
			},
			input:   "TestObject",
			target:  &inputTestObject{},
			wantErr: "graphql: the input context is empty",
		},
		{
			name:    "input is not indexed",
			ctx:     namedIndexContext,
			input:   "Absent",
			target:  &inputTestObject{},
			wantErr: `graphql: no unmarshaler for input "Absent"; inputs bound to a Go type with its own UnmarshalGQL method are not indexed`,
		},
		{
			name:    "target is not a pointer",
			ctx:     namedIndexContext,
			input:   "TestObject",
			target:  inputTestObject{},
			wantErr: "graphql: input must be a non-nil pointer",
		},
		{
			name:    "target is a nil pointer",
			ctx:     namedIndexContext,
			input:   "TestObject",
			target:  (*inputTestObject)(nil),
			wantErr: "graphql: input must be a non-nil pointer",
		},
		{
			name:    "target type is unrelated",
			ctx:     namedIndexContext,
			input:   "TestObject",
			target:  &struct{ Other int }{},
			wantErr: "graphql: cannot store unmarshaled graphql.inputTestObject in struct { Other int }",
		},
		{
			name:   "unmarshals into target",
			ctx:    namedIndexContext,
			input:  "TestObject",
			target: &inputTestObject{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalNamedInputFromContext(
				tt.ctx(t), tt.input, map[string]any{"name": "bob"}, tt.target,
			)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, inputTestObject{Name: "bob"}, *tt.target.(*inputTestObject))
		})
	}
}

// return_pointers_in_unmarshalinput decides whether a generated unmarshaler returns T or
// *T. Callers must not have to care, so either shape of target works against either
// shape of unmarshaler.
func TestUnmarshalNamedInputFromContextAbsorbsPointerConfig(t *testing.T) {
	valueCtx := WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("TestObject", unmarshalInputTestObject),
	))
	pointerCtx := WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("TestObject", unmarshalInputTestObjectPointer),
	))
	raw := map[string]any{"name": "bob"}
	want := inputTestObject{Name: "bob"}

	t.Run("value unmarshaler into value target", func(t *testing.T) {
		var got inputTestObject
		require.NoError(t, UnmarshalNamedInputFromContext(valueCtx, "TestObject", raw, &got))
		assert.Equal(t, want, got)
	})

	t.Run("value unmarshaler into pointer target", func(t *testing.T) {
		var got *inputTestObject
		require.NoError(t, UnmarshalNamedInputFromContext(valueCtx, "TestObject", raw, &got))
		require.NotNil(t, got)
		assert.Equal(t, want, *got)
	})

	t.Run("pointer unmarshaler into value target", func(t *testing.T) {
		var got inputTestObject
		require.NoError(t, UnmarshalNamedInputFromContext(pointerCtx, "TestObject", raw, &got))
		assert.Equal(t, want, got)
	})

	t.Run("pointer unmarshaler into pointer target", func(t *testing.T) {
		var got *inputTestObject
		require.NoError(t, UnmarshalNamedInputFromContext(pointerCtx, "TestObject", raw, &got))
		require.NotNil(t, got)
		assert.Equal(t, want, *got)
	})
}

func TestUnmarshalNamedInputFromContextPropagatesUnmarshalerError(t *testing.T) {
	ctx := namedIndexContext(t)

	var got inputTestObject
	err := UnmarshalNamedInputFromContext(ctx, "TestObject", "not a map", &got)

	require.EqualError(t, err, "not a map")
}

// A null input object reaches the unmarshaler as a nil any, which reflect.Call rejects
// unless it is given a typed nil.
func TestUnmarshalNamedInputFromContextNullInput(t *testing.T) {
	ctx := namedIndexContext(t)

	var got inputTestMap
	require.NoError(t, UnmarshalNamedInputFromContext(ctx, "Changes", nil, &got))

	assert.Equal(t, "changes", got["from"])
	assert.Nil(t, got["raw"])
}

func TestAssignUnmarshaled(t *testing.T) {
	object := inputTestObject{Name: "bob"}

	tests := []struct {
		name       string
		targetType reflect.Type
		result     any
		want       any
		wantErr    string
	}{
		{
			name:       "same type",
			targetType: reflect.TypeFor[inputTestObject](),
			result:     object,
			want:       object,
		},
		{
			name:       "pointer result into pointer target",
			targetType: reflect.TypeFor[*inputTestObject](),
			result:     &object,
			want:       &object,
		},
		{
			name:       "pointer result into value target",
			targetType: reflect.TypeFor[inputTestObject](),
			result:     &object,
			want:       object,
		},
		{
			name:       "nil pointer result leaves target zero",
			targetType: reflect.TypeFor[inputTestObject](),
			result:     (*inputTestObject)(nil),
			want:       inputTestObject{},
		},
		{
			name:       "value result into pointer target",
			targetType: reflect.TypeFor[*inputTestObject](),
			result:     object,
			want:       &object,
		},
		{
			name:       "unrelated types",
			targetType: reflect.TypeFor[inputTestObject](),
			result:     "bob",
			wantErr:    "graphql: cannot store unmarshaled string in graphql.inputTestObject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := reflect.New(tt.targetType).Elem()

			err := assignUnmarshaled(target, reflect.ValueOf(tt.result))

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, target.Interface())
		})
	}
}

// BindUnmarshaler adapts the free-function unmarshalers generated under
// use_function_syntax_for_execution_context, which take the execution context as an
// argument, to the shape NewInputUnmarshaler accepts.
func TestBindUnmarshaler(t *testing.T) {
	ec := &inputTestEC{}
	ctx := WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("TestObject", BindUnmarshaler(ec, unmarshalInputTestObjectWithEC)),
	))

	var got inputTestObject
	require.NoError(t, UnmarshalNamedInputFromContext(
		ctx, "TestObject", map[string]any{"name": "bob"}, &got,
	))

	assert.Equal(t, inputTestObject{Name: "bob"}, got)
	assert.Equal(t, 1, ec.calls, "the bound execution context was not passed through")
}

func TestNewInputUnmarshalerIndexLastDuplicateWins(t *testing.T) {
	ctx := WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("Shadowed", unmarshalChangesLike),
		NewInputUnmarshaler("Shadowed", unmarshalFiltersLike),
	))

	var got inputTestMap
	require.NoError(t, UnmarshalNamedInputFromContext(ctx, "Shadowed", nil, &got))

	assert.Equal(t, "filters", got["from"])
}

func TestWithLazyInputUnmarshalerIndexBuildsOnceOnFirstUse(t *testing.T) {
	var builds int
	ctx := WithLazyInputUnmarshalerIndex(context.Background(), func() *InputUnmarshalerIndex {
		builds++
		return NewInputUnmarshalerIndex(
			NewInputUnmarshaler("TestObject", unmarshalInputTestObject),
		)
	})
	assert.Zero(t, builds, "the index was built before it was needed")

	for range 3 {
		var got inputTestObject
		require.NoError(t, UnmarshalNamedInputFromContext(
			ctx, "TestObject", map[string]any{"name": "bob"}, &got,
		))
		assert.Equal(t, inputTestObject{Name: "bob"}, got)
	}
	assert.Equal(t, 1, builds)
}

// Resolvers within one request run concurrently and share its context, so the build
// function must run exactly once no matter how many of them unmarshal input.
func TestWithLazyInputUnmarshalerIndexConcurrent(t *testing.T) {
	const goroutines = 50

	// Safe to increment unguarded: the point of the test is that this runs once.
	var builds int
	ctx := WithLazyInputUnmarshalerIndex(context.Background(), func() *InputUnmarshalerIndex {
		builds++
		return NewInputUnmarshalerIndex(
			NewInputUnmarshaler("TestObject", unmarshalInputTestObject),
		)
	})

	// Each goroutine owns one slot, so results need no locking, and assertions stay on
	// the test goroutine where require is allowed to call FailNow.
	got := make([]inputTestObject, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			errs[i] = UnmarshalNamedInputFromContext(
				ctx, "TestObject", map[string]any{"name": "bob"}, &got[i],
			)
		})
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "goroutine %d", i)
		assert.Equal(t, inputTestObject{Name: "bob"}, got[i])
	}
	assert.Equal(t, 1, builds, "the index must be built exactly once per context")
}
