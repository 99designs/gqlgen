// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package subdir

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/plugin/federation/fedruntime"
	"github.com/vektah/gqlparser/v2/ast"
)

// region    ************************** generated!.gotpl **************************

// endregion ************************** generated!.gotpl **************************

// region    ***************************** args.gotpl *****************************

// endregion ***************************** args.gotpl *****************************

// region    ************************** directives.gotpl **************************

// endregion ************************** directives.gotpl **************************

// region    **************************** field.gotpl *****************************

func (ec *executionContext) __Service_sdl(ctx context.Context, field graphql.CollectedField, obj *fedruntime.Service) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext__Service_sdl(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.SDL, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	fc.Result = res
	return ec.marshalOString2string(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext__Service_sdl(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	var _ = ctx
	fc = &graphql.FieldContext{
		Object:     "_Service",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

// endregion **************************** field.gotpl *****************************

// region    **************************** input.gotpl *****************************

// endregion **************************** input.gotpl *****************************

// region    ************************** interface.gotpl ***************************

// endregion ************************** interface.gotpl ***************************

// region    **************************** object.gotpl ****************************

var _ServiceImplementors = []string{"_Service"}

func (ec *executionContext) __Service(ctx context.Context, sel ast.SelectionSet, obj *fedruntime.Service) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, _ServiceImplementors)

	out := graphql.NewFieldSet(fields)
	deferred := make(map[string]*graphql.FieldSet)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("_Service")
		case "sdl":
			out.Values[i] = ec.__Service_sdl(ctx, field, obj)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.deferred, int32(len(deferred)))

	for label, dfs := range deferred {
		ec.processDeferredGroup(graphql.DeferredGroup{
			Label:    label,
			Path:     graphql.GetPath(ctx),
			FieldSet: dfs,
			Context:  ctx,
		})
	}

	return out
}

// endregion **************************** object.gotpl ****************************

// region    ***************************** type.gotpl *****************************

func (ec *executionContext) marshalN_Service2githubᚗcomᚋ99designsᚋgqlgenᚋpluginᚋfederationᚋfedruntimeᚐService(ctx context.Context, sel ast.SelectionSet, v fedruntime.Service) graphql.Marshaler {
	var _ = ctx
	var _ = sel
	return ec.__Service(ctx, sel, &v)
}

// endregion ***************************** type.gotpl *****************************
