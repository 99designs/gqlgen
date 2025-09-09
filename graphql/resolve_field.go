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
	middlewareResolver func(ctx context.Context, next Resolver) Resolver,
	marshal func(ctx context.Context, sel ast.SelectionSet, v T) Marshaler,
	recoverFromPanic bool,
	nonNull bool,
) (ret Marshaler) {
	fc, err := initializeFieldContext(ctx, field)
	if err != nil {
		return Null
	}
	ctx = WithFieldContext(ctx, fc)
	if recoverFromPanic {
		defer func() {
			if r := recover(); r != nil {
				oc.Error(ctx, oc.Recover(ctx, r))
				ret = Null
			}
		}()
	}

	next := func(rctx context.Context) (any, error) {
		ctx = rctx // use context from middleware stack in children
		return fieldResolver(rctx)
	}

	if middlewareResolver != nil {
		next = middlewareResolver(ctx, next)
	}

	resTmp, err := oc.ResolverMiddleware(ctx, next)
	if err != nil {
		oc.Error(ctx, err)
		return Null
	}
	if resTmp == nil {
		if nonNull {
			if !HasFieldError(ctx, fc) {
				oc.Errorf(ctx, "must not be null")
			}
		}
		return Null
	}
	res := resTmp.(T)
	fc.Result = res
	return marshal(ctx, field.Selections, res)
}

func ResolveFieldStream[T any](
	ctx context.Context,
	oc *OperationContext,
	field CollectedField,
	initializeFieldContext func(ctx context.Context, field CollectedField) (*FieldContext, error),
	fieldResolver func(context.Context) (any, error),
	middlewareResolver func(ctx context.Context, next Resolver) Resolver,
	marshal func(ctx context.Context, sel ast.SelectionSet, v T) Marshaler,
	recoverFromPanic bool,
	nonNull bool,
) (ret func(ctx context.Context) Marshaler) {
	fc, err := initializeFieldContext(ctx, field)
	if err != nil {
		return nil
	}
	ctx = WithFieldContext(ctx, fc)
	if recoverFromPanic {
		defer func() {
			if r := recover(); r != nil {
				oc.Error(ctx, oc.Recover(ctx, r))
				ret = nil
			}
		}()
	}

	next := func(rctx context.Context) (any, error) {
		ctx = rctx // use context from middleware stack in children
		return fieldResolver(rctx)
	}

	if middlewareResolver != nil {
		next = middlewareResolver(ctx, next)
	}

	resTmp, err := oc.ResolverMiddleware(ctx, next)
	if err != nil {
		oc.Error(ctx, err)
		return nil
	}
	if resTmp == nil {
		if nonNull {
			if !HasFieldError(ctx, fc) {
				oc.Errorf(ctx, "must not be null")
			}
		}
		return nil
	}
	res := resTmp.(<-chan T)
	fc.Result = res
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
}
