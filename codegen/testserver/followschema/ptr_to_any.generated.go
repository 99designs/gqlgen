// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package followschema

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// region    ************************** generated!.gotpl **************************

// endregion ************************** generated!.gotpl **************************

// region    ***************************** args.gotpl *****************************

// endregion ***************************** args.gotpl *****************************

// region    ************************** directives.gotpl **************************

// endregion ************************** directives.gotpl **************************

// region    **************************** field.gotpl *****************************

func (ec *executionContext) _PtrToAnyContainer_ptrToAny(ctx context.Context, field graphql.CollectedField, obj *PtrToAnyContainer) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_PtrToAnyContainer_ptrToAny(ctx, field)
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
	resTmp := ec._fieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.PtrToAny, nil
	})

	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*any)
	fc.Result = res
	return ec.marshalOAny2ᚖinterface(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_PtrToAnyContainer_ptrToAny(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "PtrToAnyContainer",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Any does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _PtrToAnyContainer_binding(ctx context.Context, field graphql.CollectedField, obj *PtrToAnyContainer) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_PtrToAnyContainer_binding(ctx, field)
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
	resTmp := ec._fieldMiddleware(ctx, obj, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Binding(), nil
	})

	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*any)
	fc.Result = res
	return ec.marshalOAny2ᚖinterface(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_PtrToAnyContainer_binding(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "PtrToAnyContainer",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Any does not have child fields")
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

var ptrToAnyContainerImplementors = []string{"PtrToAnyContainer"}

func (ec *executionContext) _PtrToAnyContainer(ctx context.Context, sel ast.SelectionSet, obj *PtrToAnyContainer) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, ptrToAnyContainerImplementors)

	out := graphql.NewFieldSet(fields)
	deferred := make(map[string]*graphql.FieldSet)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("PtrToAnyContainer")
		case "ptrToAny":
			out.Values[i] = ec._PtrToAnyContainer_ptrToAny(ctx, field, obj)
		case "binding":
			out.Values[i] = ec._PtrToAnyContainer_binding(ctx, field, obj)
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

func (ec *executionContext) marshalNPtrToAnyContainer2githubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐPtrToAnyContainer(ctx context.Context, sel ast.SelectionSet, v PtrToAnyContainer) graphql.Marshaler {
	return ec._PtrToAnyContainer(ctx, sel, &v)
}

func (ec *executionContext) marshalNPtrToAnyContainer2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐPtrToAnyContainer(ctx context.Context, sel ast.SelectionSet, v *PtrToAnyContainer) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			ec.Errorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._PtrToAnyContainer(ctx, sel, v)
}

func (ec *executionContext) unmarshalOAny2interface(ctx context.Context, v interface{}) (any, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalAny(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOAny2interface(ctx context.Context, sel ast.SelectionSet, v any) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	res := graphql.MarshalAny(v)
	return res
}

func (ec *executionContext) unmarshalOAny2ᚖinterface(ctx context.Context, v interface{}) (*any, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalOAny2interface(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOAny2ᚖinterface(ctx context.Context, sel ast.SelectionSet, v *any) graphql.Marshaler {
	return ec.marshalOAny2interface(ctx, sel, *v)
}

// endregion ***************************** type.gotpl *****************************
