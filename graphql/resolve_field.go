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
		func(ctx context.Context, fc *FieldContext, res func() (T, error)) Marshaler {
			return WriterFunc(func(w io.Writer) {
				ret, err := res()
				fc.Result = ret
				if err != nil {
					oc.Error(ctx, err)
					Null.MarshalGQL(w)
					return
				}
				marshal(ctx, field.Selections, ret).MarshalGQL(w)
			})
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
		func(ctx context.Context, fc *FieldContext, res func() (<-chan T, error)) func(context.Context) Marshaler {
			return func(ctx context.Context) Marshaler {
				ch, err := res()
				fc.Result = ch
				if err != nil {
					oc.Error(ctx, err)
					return nil
				}

				select {
				case v, ok := <-ch:
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
	result func(ctx context.Context, fc *FieldContext, res func() (T, error)) R,
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
		resTmp = func() (T, error) {
			return res, nil
		}
	}
	res, ok := resTmp.(func() (T, error))
	if !ok {
		var t T
		oc.Errorf(ctx, `unexpected type %T from middleware/directive chain, should be %T or func() (%T, error)`, resTmp, t, t)
		return defaultResult
	}
	return result(ctx, fc, res)
}
