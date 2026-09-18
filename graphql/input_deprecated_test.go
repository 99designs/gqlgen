package graphql

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UnmarshalerMap must stay an alias, not become a defined type. Passing a plain map as an
// argument would keep compiling either way, but these two spellings only stay
// interchangeable while the types are identical.
var (
	_ func(context.Context, map[reflect.Type]reflect.Value) context.Context = WithUnmarshalerMap
	_ func(...any) map[reflect.Type]reflect.Value                           = BuildUnmarshalerMap
)

func TestUnmarshalInputFromContextWithDeprecatedMap(t *testing.T) {
	ctx := WithUnmarshalerMap(
		context.Background(),
		BuildUnmarshalerMap(unmarshalInputTestObject),
	)

	var got inputTestObject
	require.NoError(t, UnmarshalInputFromContext(ctx, map[string]any{"name": "bob"}, &got))

	assert.Equal(t, inputTestObject{Name: "bob"}, got)
}

// Generated code installs an index rather than a map, so the deprecated function has to
// keep working against one for as long as it exists.
func TestUnmarshalInputFromContextReadsTheIndex(t *testing.T) {
	ctx := WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("TestObject", unmarshalInputTestObject),
	))

	var got inputTestObject
	require.NoError(t, UnmarshalInputFromContext(ctx, map[string]any{"name": "bob"}, &got))

	assert.Equal(t, inputTestObject{Name: "bob"}, got)
}

// Selecting by Go type cannot distinguish inputs that share one. Reporting that beats
// silently returning whichever unmarshaler happened to be registered last.
func TestUnmarshalInputFromContextReportsSharedGoType(t *testing.T) {
	ctx := WithInputUnmarshalerIndex(context.Background(), NewInputUnmarshalerIndex(
		NewInputUnmarshaler("Changes", unmarshalChangesLike),
		NewInputUnmarshaler("Filters", unmarshalFiltersLike),
	))

	var got inputTestMap
	err := UnmarshalInputFromContext(ctx, nil, &got)

	require.EqualError(t, err,
		"graphql: 2 input types unmarshal to map[string]interface {} (Changes, Filters); "+
			"use UnmarshalNamedInputFromContext")
}

func TestUnmarshalInputFromContextErrors(t *testing.T) {
	built := BuildUnmarshalerMap(unmarshalInputTestObject)

	tests := []struct {
		name    string
		ctx     func() context.Context
		target  any
		wantErr string
	}{
		{
			name:    "no unmarshaler map in context",
			ctx:     context.Background,
			target:  &inputTestObject{},
			wantErr: "graphql: the input context is empty",
		},
		{
			name: "nil build function",
			ctx: func() context.Context {
				return WithLazyUnmarshalerMap(context.Background(), nil)
			},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalInputFromContext(tt.ctx(), map[string]any{"name": "bob"}, tt.target)

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestWithLazyUnmarshalerMapBuildsOnceOnFirstUse(t *testing.T) {
	var builds int
	ctx := WithLazyUnmarshalerMap(context.Background(), func() UnmarshalerMap {
		builds++
		return BuildUnmarshalerMap(unmarshalInputTestObject)
	})
	assert.Zero(t, builds, "the map was built before it was needed")

	for range 3 {
		var got inputTestObject
		require.NoError(t, UnmarshalInputFromContext(ctx, map[string]any{"name": "bob"}, &got))
	}
	assert.Equal(t, 1, builds)
}

// An unmarshaler that still takes its execution context explicitly cannot be invoked with
// (ctx, raw), so storing it would panic inside reflect at call time.
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
