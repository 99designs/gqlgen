package graphql

// This file holds the input-unmarshaling API that predates
// InputUnmarshalerIndex. It is kept so that code written against the shipped
// surface keeps compiling, and is intended to be deleted whole once the
// deprecation window closes. Nothing generated calls into it.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
)

const unmarshalInputCtx key = "unmarshal_input_context"

// UnmarshalerMap maps each input type to the generated function that unmarshals it.
//
// Deprecated: entries carry no GraphQL name, so two input objects that unmarshal to the
// same Go type cannot be told apart. Use [InputUnmarshalerIndex] instead.
type UnmarshalerMap = map[reflect.Type]reflect.Value

// BuildUnmarshalerMap returns a map of unmarshal functions of the ExecutableContext
// to use with the WithUnmarshalerMap function.
//
// Entries are invoked as func(context.Context, any), so arguments that are not functions
// of that shape are skipped rather than stored.
//
// Deprecated: use [NewInputUnmarshalerIndex], which keys by GraphQL input name and so
// can represent two inputs that share a Go type.
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

// WithUnmarshalerMap returns a new context with a map from input types to their unmarshaler
// functions.
//
// Deprecated: use [WithInputUnmarshalerIndex].
func WithUnmarshalerMap(ctx context.Context, maps UnmarshalerMap) context.Context {
	// maps is already built, so there is nothing left to defer.
	return context.WithValue(ctx, unmarshalInputCtx, func() UnmarshalerMap { return maps })
}

// WithLazyUnmarshalerMap returns a new context that builds its map from input types to
// unmarshaler functions on first use instead of up front.
//
// build is called at most once per returned context; concurrent callers share the one
// result. A nil build installs nothing, leaving ctx as it was.
//
// Deprecated: use [WithLazyInputUnmarshalerIndex].
func WithLazyUnmarshalerMap(ctx context.Context, build func() UnmarshalerMap) context.Context {
	if build == nil {
		return ctx
	}
	return context.WithValue(ctx, unmarshalInputCtx, sync.OnceValue(build))
}

// UnmarshalInputFromContext allows unmarshaling input object from a context.
//
// The input object is chosen by the Go type v points at, which cannot distinguish two
// GraphQL inputs that unmarshal to the same Go type — every map-backed input unmarshals
// to map[string]any, for instance. When the type is shared this reports an error naming
// the candidates rather than picking one.
//
// Deprecated: use [UnmarshalNamedInputFromContext], which selects by GraphQL input name
// and is never ambiguous.
func UnmarshalInputFromContext(ctx context.Context, raw, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("graphql: input must be a non-nil pointer")
	}
	target := rv.Elem().Type()

	// Generated code installs an index; only hand-written callers still install a map.
	if idx := inputUnmarshalerIndexFromContext(ctx); idx != nil {
		return unmarshalByGoType(ctx, idx, target, raw, v)
	}

	var m UnmarshalerMap
	if build, _ := ctx.Value(unmarshalInputCtx).(func() UnmarshalerMap); build != nil {
		m = build()
	}
	if m == nil {
		return errEmptyInputContext
	}

	fn, ok := m[target]
	if !ok {
		return errors.New("graphql: no unmarshal function found")
	}
	return InputUnmarshaler{goType: target, fn: fn}.unmarshal(ctx, raw, v)
}

// unmarshalByGoType resolves an input by the Go type its unmarshaler returns. The index
// is keyed by name, so this scans it; the scan is confined to this deprecated path and
// lets the ambiguity that the Go type cannot resolve be reported precisely.
func unmarshalByGoType(
	ctx context.Context,
	idx *InputUnmarshalerIndex,
	target reflect.Type,
	raw, v any,
) error {
	names := idx.namesForGoType(target)
	switch len(names) {
	case 0:
		return errors.New("graphql: no unmarshal function found")
	case 1:
		return idx.byName[names[0]].unmarshal(ctx, raw, v)
	default:
		return fmt.Errorf(
			"graphql: %d input types unmarshal to %s (%s); use UnmarshalNamedInputFromContext",
			len(names), target, strings.Join(names, ", "),
		)
	}
}

// namesForGoType returns the GraphQL names of every indexed input whose unmarshaler
// returns t, sorted so that error messages are stable. Only the type-keyed lookup above
// needs it, because the index itself is keyed by name.
func (idx *InputUnmarshalerIndex) namesForGoType(t reflect.Type) []string {
	var names []string
	for name, u := range idx.byName {
		if u.goType == t {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}
