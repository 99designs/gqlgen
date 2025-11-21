package graphql

import (
	"context"
	"io"

	"github.com/vektah/gqlparser/v2/ast"
)

func ResolveField[T any](
	ctx context.Context,
	oc *OperationContext,
	field CollectedField,
	initializeFieldContext func(ctx context.Context, field CollectedField) (*FieldContext, error),
	fieldResolver func(ctx context.Context) (any, error),
	middlewareChain func(ctx context.Context, next Resolver) Resolver,
	marshal func(ctx context.Context, sel ast.SelectionSet, v T) Marshaler,
	recoverFromPanic bool,
	nonNull bool,
) Marshaler {
	return resolveField[T, Marshaler](
		ctx,
		oc,
		field,
		initializeFieldContext,
		fieldResolver,
		middlewareChain,
		recoverFromPanic,
		nonNull,
		Null,
		func(ctx context.Context, res T) Marshaler {
			return marshal(ctx, field.Selections, res)
		},
	)
}

func ResolveFieldStream[T any](
	ctx context.Context,
	oc *OperationContext,
	field CollectedField,
	initializeFieldContext func(ctx context.Context, field CollectedField) (*FieldContext, error),
	fieldResolver func(context.Context) (any, error),
	middlewareChain func(ctx context.Context, next Resolver) Resolver,
	marshal func(ctx context.Context, sel ast.SelectionSet, v T) Marshaler,
	recoverFromPanic bool,
	nonNull bool,
) func(context.Context) Marshaler {
	return resolveField(
		ctx,
		oc,
		field,
		initializeFieldContext,
		fieldResolver,
		middlewareChain,
		recoverFromPanic,
		nonNull,
		nil,
		func(ctx context.Context, res <-chan T) func(context.Context) Marshaler {
			return func(ctx context.Context) Marshaler {
				select {
				case v, ok := <-res:
					if !ok {
						return nil
					}
					return WriterFunc(func(w io.Writer) {
						w.Write([]byte{'{'})
						MarshalString(field.Alias).MarshalGQL(w)
						w.Write([]byte{':'})
						marshal(ctx, field.Selections, v).MarshalGQL(w)
						w.Write([]byte{'}'})
					})
				case <-ctx.Done():
					return nil
				}
			}
		},
	)
}

func resolveField[T, R any](
	ctx context.Context,
	oc *OperationContext,
	field CollectedField,
	initializeFieldContext func(ctx context.Context, field CollectedField) (*FieldContext, error),
	fieldResolver func(ctx context.Context) (any, error),
	middlewareChain func(ctx context.Context, next Resolver) Resolver,
	recoverFromPanic bool,
	nonNull bool,
	defaultResult R,
	result func(ctx context.Context, res T) R,
) (ret R) {
	fc, err := initializeFieldContext(ctx, field)
	if err != nil {
		return defaultResult
	}
	ctx = WithFieldContext(ctx, fc)

	if recoverFromPanic {
		defer func() {
			if r := recover(); r != nil {
				oc.Error(ctx, oc.Recover(ctx, r))
				ret = defaultResult
			}
		}()
	}

	next := func(rctx context.Context) (any, error) {
		ctx = rctx // use context from middleware stack in children
		return fieldResolver(rctx)
	}

	if middlewareChain != nil {
		next = middlewareChain(ctx, next)
	}

	resTmp, err := oc.ResolverMiddleware(ctx, next)
	if err != nil {
		oc.Error(ctx, err)
		return defaultResult
	}
	if resTmp == nil {
		if nonNull {
			if !HasFieldError(ctx, fc) {
				oc.Errorf(ctx, "must not be null")
			}
		}
		return defaultResult
	}
	if res, ok := resTmp.(T); ok {
		fc.Result = res
		return result(ctx, res)
	}
	if res, ok := resTmp.(R); ok {
		fc.Result = res
		return res
	}
	var t T
	oc.Errorf(
		ctx,
		`unexpected type %T from middleware/directive chain, should be %T`,
		resTmp,
		t,
	)
	return defaultResult
}
