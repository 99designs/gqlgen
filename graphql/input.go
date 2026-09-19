package graphql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

const inputUnmarshalerIndexCtx key = "input_unmarshaler_index_context"

// errEmptyInputContext reports a context that never passed through a gqlgen request, so
// carries no unmarshalers.
var errEmptyInputContext = errors.New("graphql: the input context is empty")

// InputUnmarshaler is one generated input-object unmarshaler together with the GraphQL
// input name it was generated for. Generated code builds these; application code has no
// reason to construct one directly.
type InputUnmarshaler struct {
	name string        // GraphQL input object name, e.g. "SearchFilters"
	fn   reflect.Value // func(context.Context, any) (T, error)

	// goType is what unmarshal returns, T or *T. Only the deprecated type-keyed lookup
	// needs it; it can go when input_deprecated.go does.
	goType reflect.Type
}

// NewInputUnmarshaler pairs a generated unmarshaler with the name of the GraphQL input
// object it unmarshals. The generic signature records T without reflecting over
// unmarshal, and makes a wrongly shaped unmarshaler a compile error rather than a panic
// at call time.
func NewInputUnmarshaler[T any](
	name string,
	unmarshal func(ctx context.Context, obj any) (T, error),
) InputUnmarshaler {
	return InputUnmarshaler{
		name:   name,
		goType: reflect.TypeFor[T](),
		fn:     reflect.ValueOf(unmarshal),
	}
}

// InputUnmarshalerIndex resolves a GraphQL input object name to the unmarshaler
// generated for it.
//
// One index belongs to one request: the unmarshalers it holds are bound to that
// request's execution context, so an index must not be shared between requests.
type InputUnmarshalerIndex struct {
	byName map[string]InputUnmarshaler
}

// NewInputUnmarshalerIndex indexes unmarshalers by their GraphQL input name.
//
// GraphQL type names are unique within a schema, so generated code cannot produce a
// duplicate. If one is passed anyway the last wins, which leaves the constructor with no
// failure mode for callers to handle.
func NewInputUnmarshalerIndex(unmarshalers ...InputUnmarshaler) *InputUnmarshalerIndex {
	byName := make(map[string]InputUnmarshaler, len(unmarshalers))
	for _, u := range unmarshalers {
		byName[u.name] = u
	}
	return &InputUnmarshalerIndex{byName: byName}
}

// WithInputUnmarshalerIndex returns a context carrying idx, so that resolvers reached
// from it can unmarshal input objects by name.
func WithInputUnmarshalerIndex(ctx context.Context, idx *InputUnmarshalerIndex) context.Context {
	// idx is already built, so there is nothing left to defer.
	return context.WithValue(
		ctx,
		inputUnmarshalerIndexCtx,
		func() *InputUnmarshalerIndex { return idx },
	)
}

// WithLazyInputUnmarshalerIndex returns a context that builds its index on first use
// instead of up front.
//
// Generated unmarshalers are bound to a single request's execution context, so the index
// cannot be shared between requests and building it costs one bound method value per
// input type. Most requests never unmarshal an input object this way, so deferring the
// build keeps that cost off the common path.
//
// build is called at most once per returned context, on the first lookup; concurrent
// callers share the one result. A nil build installs nothing, leaving ctx as it was.
func WithLazyInputUnmarshalerIndex(
	ctx context.Context,
	build func() *InputUnmarshalerIndex,
) context.Context {
	if build == nil {
		return ctx
	}
	return context.WithValue(ctx, inputUnmarshalerIndexCtx, sync.OnceValue(build))
}

func inputUnmarshalerIndexFromContext(ctx context.Context) *InputUnmarshalerIndex {
	build, _ := ctx.Value(inputUnmarshalerIndexCtx).(func() *InputUnmarshalerIndex)
	if build == nil {
		return nil
	}
	return build()
}

// BindUnmarshaler binds ec to an unmarshaler that takes its execution context as an
// argument, yielding the two-argument shape NewInputUnmarshaler accepts.
//
// The use_function_syntax_for_execution_context config option generates unmarshalers as
// free functions taking *executionContext rather than as methods on it, so that very
// large schemas stay under Go's per-type method limit. Those functions cannot be indexed
// directly.
func BindUnmarshaler[EC, T any](
	ec EC,
	unmarshal func(ctx context.Context, ec EC, obj any) (T, error),
) func(ctx context.Context, obj any) (T, error) {
	return func(ctx context.Context, obj any) (T, error) {
		return unmarshal(ctx, ec, obj)
	}
}

// UnmarshalNamedInputFromContext unmarshals raw into v using the unmarshaler generated
// for the GraphQL input object called name.
//
// v may point at the input's Go type or at a pointer to it. Which one the generated
// unmarshaler returns depends on the schema's return_pointers_in_unmarshalinput setting,
// and callers do not need to know which was chosen.
//
// Inputs bound to a Go type that supplies its own UnmarshalGQL method are not indexed,
// because no unmarshaler is generated for them.
func UnmarshalNamedInputFromContext(ctx context.Context, name string, raw, v any) error {
	idx := inputUnmarshalerIndexFromContext(ctx)
	if idx == nil {
		return errEmptyInputContext
	}

	u, ok := idx.byName[name]
	if !ok {
		return fmt.Errorf(
			"graphql: no unmarshaler for input %q; inputs bound to a Go type with its own UnmarshalGQL method are not indexed",
			name,
		)
	}

	return u.unmarshal(ctx, raw, v)
}

// unmarshal calls the generated unmarshaler and stores the result through v, which must
// be a non-nil pointer to the input's Go type or to a pointer to it.
func (u InputUnmarshaler) unmarshal(ctx context.Context, raw, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("graphql: input must be a non-nil pointer")
	}

	// reflect.ValueOf(nil) is the zero Value, which Call rejects. A null input object is
	// ordinary, and the generated unmarshalers handle it, so pass a typed nil instead.
	rawValue := reflect.ValueOf(raw)
	if !rawValue.IsValid() {
		rawValue = reflect.Zero(u.fn.Type().In(1))
	}

	res := u.fn.Call([]reflect.Value{reflect.ValueOf(ctx), rawValue})
	if err := res[1].Interface(); err != nil {
		return err.(error)
	}

	return assignUnmarshaled(rv.Elem(), res[0])
}

// assignUnmarshaled stores result in target, adding or removing one level of indirection
// if the two differ by exactly that. This is what keeps
// return_pointers_in_unmarshalinput from reaching call sites: the generated unmarshaler
// returns T or *T according to that setting, while the caller passes whichever of the two
// it finds convenient.
//
// A nil pointer result leaves target at its zero value, so both settings agree on what a
// null input produces.
func assignUnmarshaled(target, result reflect.Value) error {
	switch {
	case result.Type().AssignableTo(target.Type()):
		target.Set(result)

	case result.Kind() == reflect.Pointer && result.Type().Elem().AssignableTo(target.Type()):
		if result.IsNil() {
			target.SetZero()
			return nil
		}
		target.Set(result.Elem())

	case target.Kind() == reflect.Pointer && result.Type().AssignableTo(target.Type().Elem()):
		held := reflect.New(target.Type().Elem())
		held.Elem().Set(result)
		target.Set(held)

	default:
		return fmt.Errorf(
			"graphql: cannot store unmarshaled %s in %s",
			result.Type(), target.Type(),
		)
	}
	return nil
}
