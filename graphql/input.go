package graphql

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

const unmarshalInputCtx key = "unmarshal_input_context"

// UnmarshalerMap maps each input type to the generated function that unmarshals it.
type UnmarshalerMap = map[reflect.Type]reflect.Value

// BuildUnmarshalerMap returns a map of unmarshal functions of the ExecutableContext
// to use with the WithUnmarshalerMap function.
//
// UnmarshalInputFromContext invokes each entry as func(context.Context, any), so
// arguments that are not functions of that shape are skipped rather than stored.
// Generated unmarshalers that take the execution context explicitly need
// BindUnmarshaler to reach that shape.
func BuildUnmarshalerMap(unmarshaler ...any) UnmarshalerMap {
	maps := make(UnmarshalerMap)
	for _, v := range unmarshaler {
		ft := reflect.TypeOf(v)
		if ft.Kind() == reflect.Func && ft.NumIn() == 2 && ft.NumOut() == 2 {
			maps[ft.Out(0)] = reflect.ValueOf(v)
		}
	}

	return maps
}

// BindUnmarshaler binds ec to an unmarshaler that takes its execution context as an
// argument, yielding the two-argument shape BuildUnmarshalerMap stores.
//
// The use_function_syntax_for_execution_context config option generates unmarshalers as
// free functions taking *executionContext rather than as methods on it, so that very
// large schemas stay under Go's per-type method limit. Those functions cannot go into an
// UnmarshalerMap directly.
func BindUnmarshaler[EC, T any](
	ec EC,
	unmarshal func(ctx context.Context, ec EC, obj any) (T, error),
) func(ctx context.Context, obj any) (T, error) {
	return func(ctx context.Context, obj any) (T, error) {
		return unmarshal(ctx, ec, obj)
	}
}

// WithUnmarshalerMap returns a new context with a map from input types to their unmarshaler
// functions.
func WithUnmarshalerMap(ctx context.Context, maps UnmarshalerMap) context.Context {
	// maps is already built, so there is nothing left to defer.
	return context.WithValue(ctx, unmarshalInputCtx, func() UnmarshalerMap { return maps })
}

// WithLazyUnmarshalerMap returns a new context that builds its map from input types to
// unmarshaler functions on first use instead of up front.
//
// Generated unmarshalers are methods bound to a single request's execution context, so
// the map cannot be shared between requests and building it costs one bound method value
// per input type. Most requests never reach UnmarshalInputFromContext, so deferring the
// build keeps that cost off the common path.
//
// build is called at most once per returned context, on the first call to
// UnmarshalInputFromContext; concurrent callers share the one result. A nil build
// installs nothing, leaving ctx as it was.
func WithLazyUnmarshalerMap(ctx context.Context, build func() UnmarshalerMap) context.Context {
	if build == nil {
		return ctx
	}
	return context.WithValue(ctx, unmarshalInputCtx, sync.OnceValue(build))
}

// UnmarshalInputFromContext allows unmarshaling input object from a context.
func UnmarshalInputFromContext(ctx context.Context, raw, v any) error {
	var m UnmarshalerMap
	if build, _ := ctx.Value(unmarshalInputCtx).(func() UnmarshalerMap); build != nil {
		m = build()
	}
	if m == nil {
		return errors.New("graphql: the input context is empty")
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("graphql: input must be a non-nil pointer")
	}
	if fn, ok := m[rv.Elem().Type()]; ok {
		res := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(raw),
		})
		if err := res[1].Interface(); err != nil {
			return err.(error)
		}

		rv.Elem().Set(res[0])
		return nil
	}

	return errors.New("graphql: no unmarshal function found")
}
