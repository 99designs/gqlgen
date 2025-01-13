// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package followschema

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// region    ************************** generated!.gotpl **************************

// endregion ************************** generated!.gotpl **************************

// region    ***************************** args.gotpl *****************************

func (ec *executionContext) dir_length_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := ec.dir_length_argsMin(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["min"] = arg0
	arg1, err := ec.dir_length_argsMax(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["max"] = arg1
	arg2, err := ec.dir_length_argsMessage(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["message"] = arg2
	return args, nil
}
func (ec *executionContext) dir_length_argsMin(
	ctx context.Context,
	rawArgs map[string]any,
) (int, error) {
	if _, ok := rawArgs["min"]; !ok {
		var zeroVal int
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("min"))
	if tmp, ok := rawArgs["min"]; ok {
		return ec.unmarshalNInt2int(ctx, tmp)
	}

	var zeroVal int
	return zeroVal, nil
}

func (ec *executionContext) dir_length_argsMax(
	ctx context.Context,
	rawArgs map[string]any,
) (*int, error) {
	if _, ok := rawArgs["max"]; !ok {
		var zeroVal *int
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("max"))
	if tmp, ok := rawArgs["max"]; ok {
		return ec.unmarshalOInt2ᚖint(ctx, tmp)
	}

	var zeroVal *int
	return zeroVal, nil
}

func (ec *executionContext) dir_length_argsMessage(
	ctx context.Context,
	rawArgs map[string]any,
) (*string, error) {
	if _, ok := rawArgs["message"]; !ok {
		var zeroVal *string
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("message"))
	if tmp, ok := rawArgs["message"]; ok {
		return ec.unmarshalOString2ᚖstring(ctx, tmp)
	}

	var zeroVal *string
	return zeroVal, nil
}

func (ec *executionContext) dir_logged_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := ec.dir_logged_argsID(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["id"] = arg0
	return args, nil
}
func (ec *executionContext) dir_logged_argsID(
	ctx context.Context,
	rawArgs map[string]any,
) (string, error) {
	if _, ok := rawArgs["id"]; !ok {
		var zeroVal string
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("id"))
	if tmp, ok := rawArgs["id"]; ok {
		return ec.unmarshalNUUID2string(ctx, tmp)
	}

	var zeroVal string
	return zeroVal, nil
}

func (ec *executionContext) dir_order1_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := ec.dir_order1_argsLocation(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["location"] = arg0
	return args, nil
}
func (ec *executionContext) dir_order1_argsLocation(
	ctx context.Context,
	rawArgs map[string]any,
) (string, error) {
	if _, ok := rawArgs["location"]; !ok {
		var zeroVal string
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("location"))
	if tmp, ok := rawArgs["location"]; ok {
		return ec.unmarshalNString2string(ctx, tmp)
	}

	var zeroVal string
	return zeroVal, nil
}

func (ec *executionContext) dir_order2_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := ec.dir_order2_argsLocation(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["location"] = arg0
	return args, nil
}
func (ec *executionContext) dir_order2_argsLocation(
	ctx context.Context,
	rawArgs map[string]any,
) (string, error) {
	if _, ok := rawArgs["location"]; !ok {
		var zeroVal string
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("location"))
	if tmp, ok := rawArgs["location"]; ok {
		return ec.unmarshalNString2string(ctx, tmp)
	}

	var zeroVal string
	return zeroVal, nil
}

func (ec *executionContext) dir_populate_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := ec.dir_populate_argsValue(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["value"] = arg0
	return args, nil
}
func (ec *executionContext) dir_populate_argsValue(
	ctx context.Context,
	rawArgs map[string]any,
) (string, error) {
	if _, ok := rawArgs["value"]; !ok {
		var zeroVal string
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("value"))
	if tmp, ok := rawArgs["value"]; ok {
		return ec.unmarshalNString2string(ctx, tmp)
	}

	var zeroVal string
	return zeroVal, nil
}

func (ec *executionContext) dir_range_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := ec.dir_range_argsMin(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["min"] = arg0
	arg1, err := ec.dir_range_argsMax(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	args["max"] = arg1
	return args, nil
}
func (ec *executionContext) dir_range_argsMin(
	ctx context.Context,
	rawArgs map[string]any,
) (*int, error) {
	if _, ok := rawArgs["min"]; !ok {
		var zeroVal *int
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("min"))
	if tmp, ok := rawArgs["min"]; ok {
		return ec.unmarshalOInt2ᚖint(ctx, tmp)
	}

	var zeroVal *int
	return zeroVal, nil
}

func (ec *executionContext) dir_range_argsMax(
	ctx context.Context,
	rawArgs map[string]any,
) (*int, error) {
	if _, ok := rawArgs["max"]; !ok {
		var zeroVal *int
		return zeroVal, nil
	}

	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("max"))
	if tmp, ok := rawArgs["max"]; ok {
		return ec.unmarshalOInt2ᚖint(ctx, tmp)
	}

	var zeroVal *int
	return zeroVal, nil
}

// endregion ***************************** args.gotpl *****************************

// region    ************************** directives.gotpl **************************

func (ec *executionContext) _fieldMiddleware(ctx context.Context, obj any, next graphql.Resolver) any {
	fc := graphql.GetFieldContext(ctx)
	for _, d := range fc.Field.Directives {
		switch d.Name {
		case "logged":
			rawArgs := d.ArgumentMap(ec.Variables)
			args, err := ec.dir_logged_args(ctx, rawArgs)
			if err != nil {
				ec.Error(ctx, err)
				return nil
			}
			n := next
			next = func(ctx context.Context) (any, error) {
				if ec.directives.Logged == nil {
					return nil, errors.New("directive logged is not implemented")
				}
				return ec.directives.Logged(ctx, obj, n, args["id"].(string))
			}
		}
	}
	res, err := ec.ResolverMiddleware(ctx, next)
	if err != nil {
		ec.Error(ctx, err)
		return nil
	}
	return res
}

// endregion ************************** directives.gotpl **************************

// region    **************************** field.gotpl *****************************

func (ec *executionContext) _ObjectDirectives_text(ctx context.Context, field graphql.CollectedField, obj *ObjectDirectives) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_ObjectDirectives_text(ctx, field)
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
	resTmp := ec._fieldMiddleware(ctx, obj, func(rctx context.Context) (any, error) {
		directive0 := func(rctx context.Context) (any, error) {
			ctx = rctx // use context from middleware stack in children
			return obj.Text, nil
		}

		directive1 := func(ctx context.Context) (any, error) {
			min, err := ec.unmarshalNInt2int(ctx, 0)
			if err != nil {
				var zeroVal string
				return zeroVal, err
			}
			max, err := ec.unmarshalOInt2ᚖint(ctx, 7)
			if err != nil {
				var zeroVal string
				return zeroVal, err
			}
			message, err := ec.unmarshalOString2ᚖstring(ctx, "not valid")
			if err != nil {
				var zeroVal string
				return zeroVal, err
			}
			if ec.directives.Length == nil {
				var zeroVal string
				return zeroVal, errors.New("directive length is not implemented")
			}
			return ec.directives.Length(ctx, obj, directive0, min, max, message)
		}

		tmp, err := directive1(rctx)
		if err != nil {
			return nil, graphql.ErrorOnPath(ctx, err)
		}
		if tmp == nil {
			return nil, nil
		}
		if data, ok := tmp.(string); ok {
			return data, nil
		}
		return nil, fmt.Errorf(`unexpected type %T from directive, should be string`, tmp)
	})

	if resTmp == nil {
		if !graphql.HasFieldError(ctx, fc) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	fc.Result = res
	return ec.marshalNString2string(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_ObjectDirectives_text(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "ObjectDirectives",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _ObjectDirectives_nullableText(ctx context.Context, field graphql.CollectedField, obj *ObjectDirectives) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_ObjectDirectives_nullableText(ctx, field)
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
	resTmp := ec._fieldMiddleware(ctx, obj, func(rctx context.Context) (any, error) {
		directive0 := func(rctx context.Context) (any, error) {
			ctx = rctx // use context from middleware stack in children
			return obj.NullableText, nil
		}

		directive1 := func(ctx context.Context) (any, error) {
			if ec.directives.ToNull == nil {
				var zeroVal *string
				return zeroVal, errors.New("directive toNull is not implemented")
			}
			return ec.directives.ToNull(ctx, obj, directive0)
		}

		tmp, err := directive1(rctx)
		if err != nil {
			return nil, graphql.ErrorOnPath(ctx, err)
		}
		if tmp == nil {
			return nil, nil
		}
		if data, ok := tmp.(*string); ok {
			return data, nil
		}
		return nil, fmt.Errorf(`unexpected type %T from directive, should be *string`, tmp)
	})

	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	fc.Result = res
	return ec.marshalOString2ᚖstring(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_ObjectDirectives_nullableText(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "ObjectDirectives",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _ObjectDirectives_order(ctx context.Context, field graphql.CollectedField, obj *ObjectDirectives) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_ObjectDirectives_order(ctx, field)
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
	resTmp := ec._fieldMiddleware(ctx, obj, func(rctx context.Context) (any, error) {
		ctx = rctx // use context from middleware stack in children
		return obj.Order, nil
	})

	if resTmp == nil {
		if !graphql.HasFieldError(ctx, fc) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.([]string)
	fc.Result = res
	return ec.marshalNString2ᚕstringᚄ(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_ObjectDirectives_order(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "ObjectDirectives",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _ObjectDirectivesWithCustomGoModel_nullableText(ctx context.Context, field graphql.CollectedField, obj *ObjectDirectivesWithCustomGoModel) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_ObjectDirectivesWithCustomGoModel_nullableText(ctx, field)
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
	resTmp := ec._fieldMiddleware(ctx, obj, func(rctx context.Context) (any, error) {
		directive0 := func(rctx context.Context) (any, error) {
			ctx = rctx // use context from middleware stack in children
			return obj.NullableText, nil
		}

		directive1 := func(ctx context.Context) (any, error) {
			if ec.directives.ToNull == nil {
				var zeroVal string
				return zeroVal, errors.New("directive toNull is not implemented")
			}
			return ec.directives.ToNull(ctx, obj, directive0)
		}

		tmp, err := directive1(rctx)
		if err != nil {
			return nil, graphql.ErrorOnPath(ctx, err)
		}
		if tmp == nil {
			return nil, nil
		}
		if data, ok := tmp.(string); ok {
			return data, nil
		}
		return nil, fmt.Errorf(`unexpected type %T from directive, should be string`, tmp)
	})

	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(string)
	fc.Result = res
	return ec.marshalOString2string(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_ObjectDirectivesWithCustomGoModel_nullableText(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "ObjectDirectivesWithCustomGoModel",
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

func (ec *executionContext) unmarshalInputInnerDirectives(ctx context.Context, obj any) (InnerDirectives, error) {
	var it InnerDirectives
	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"message"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "message":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("message"))
			directive0 := func(ctx context.Context) (any, error) { return ec.unmarshalNString2string(ctx, v) }

			directive1 := func(ctx context.Context) (any, error) {
				min, err := ec.unmarshalNInt2int(ctx, 1)
				if err != nil {
					var zeroVal string
					return zeroVal, err
				}
				message, err := ec.unmarshalOString2ᚖstring(ctx, "not valid")
				if err != nil {
					var zeroVal string
					return zeroVal, err
				}
				if ec.directives.Length == nil {
					var zeroVal string
					return zeroVal, errors.New("directive length is not implemented")
				}
				return ec.directives.Length(ctx, obj, directive0, min, nil, message)
			}

			tmp, err := directive1(ctx)
			if err != nil {
				return it, graphql.ErrorOnPath(ctx, err)
			}
			if data, ok := tmp.(string); ok {
				it.Message = data
			} else {
				err := fmt.Errorf(`unexpected type %T from directive, should be string`, tmp)
				return it, graphql.ErrorOnPath(ctx, err)
			}
		}
	}

	return it, nil
}

func (ec *executionContext) unmarshalInputInputDirectives(ctx context.Context, obj any) (InputDirectives, error) {
	var it InputDirectives
	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"text", "nullableText", "inner", "innerNullable", "thirdParty"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "text":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("text"))
			directive0 := func(ctx context.Context) (any, error) { return ec.unmarshalNString2string(ctx, v) }

			directive1 := func(ctx context.Context) (any, error) {
				if ec.directives.Directive3 == nil {
					var zeroVal string
					return zeroVal, errors.New("directive directive3 is not implemented")
				}
				return ec.directives.Directive3(ctx, obj, directive0)
			}
			directive2 := func(ctx context.Context) (any, error) {
				min, err := ec.unmarshalNInt2int(ctx, 0)
				if err != nil {
					var zeroVal string
					return zeroVal, err
				}
				max, err := ec.unmarshalOInt2ᚖint(ctx, 7)
				if err != nil {
					var zeroVal string
					return zeroVal, err
				}
				message, err := ec.unmarshalOString2ᚖstring(ctx, "not valid")
				if err != nil {
					var zeroVal string
					return zeroVal, err
				}
				if ec.directives.Length == nil {
					var zeroVal string
					return zeroVal, errors.New("directive length is not implemented")
				}
				return ec.directives.Length(ctx, obj, directive1, min, max, message)
			}

			tmp, err := directive2(ctx)
			if err != nil {
				return it, graphql.ErrorOnPath(ctx, err)
			}
			if data, ok := tmp.(string); ok {
				it.Text = data
			} else {
				err := fmt.Errorf(`unexpected type %T from directive, should be string`, tmp)
				return it, graphql.ErrorOnPath(ctx, err)
			}
		case "nullableText":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("nullableText"))
			directive0 := func(ctx context.Context) (any, error) { return ec.unmarshalOString2ᚖstring(ctx, v) }

			directive1 := func(ctx context.Context) (any, error) {
				if ec.directives.Directive3 == nil {
					var zeroVal *string
					return zeroVal, errors.New("directive directive3 is not implemented")
				}
				return ec.directives.Directive3(ctx, obj, directive0)
			}
			directive2 := func(ctx context.Context) (any, error) {
				if ec.directives.ToNull == nil {
					var zeroVal *string
					return zeroVal, errors.New("directive toNull is not implemented")
				}
				return ec.directives.ToNull(ctx, obj, directive1)
			}

			tmp, err := directive2(ctx)
			if err != nil {
				return it, graphql.ErrorOnPath(ctx, err)
			}
			if data, ok := tmp.(*string); ok {
				it.NullableText = data
			} else if tmp == nil {
				it.NullableText = nil
			} else {
				err := fmt.Errorf(`unexpected type %T from directive, should be *string`, tmp)
				return it, graphql.ErrorOnPath(ctx, err)
			}
		case "inner":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("inner"))
			directive0 := func(ctx context.Context) (any, error) {
				return ec.unmarshalNInnerDirectives2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐInnerDirectives(ctx, v)
			}

			directive1 := func(ctx context.Context) (any, error) {
				if ec.directives.Directive3 == nil {
					var zeroVal *InnerDirectives
					return zeroVal, errors.New("directive directive3 is not implemented")
				}
				return ec.directives.Directive3(ctx, obj, directive0)
			}

			tmp, err := directive1(ctx)
			if err != nil {
				return it, graphql.ErrorOnPath(ctx, err)
			}
			if data, ok := tmp.(*InnerDirectives); ok {
				it.Inner = data
			} else if tmp == nil {
				it.Inner = nil
			} else {
				err := fmt.Errorf(`unexpected type %T from directive, should be *github.com/99designs/gqlgen/codegen/testserver/followschema.InnerDirectives`, tmp)
				return it, graphql.ErrorOnPath(ctx, err)
			}
		case "innerNullable":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("innerNullable"))
			directive0 := func(ctx context.Context) (any, error) {
				return ec.unmarshalOInnerDirectives2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐInnerDirectives(ctx, v)
			}

			directive1 := func(ctx context.Context) (any, error) {
				if ec.directives.Directive3 == nil {
					var zeroVal *InnerDirectives
					return zeroVal, errors.New("directive directive3 is not implemented")
				}
				return ec.directives.Directive3(ctx, obj, directive0)
			}

			tmp, err := directive1(ctx)
			if err != nil {
				return it, graphql.ErrorOnPath(ctx, err)
			}
			if data, ok := tmp.(*InnerDirectives); ok {
				it.InnerNullable = data
			} else if tmp == nil {
				it.InnerNullable = nil
			} else {
				err := fmt.Errorf(`unexpected type %T from directive, should be *github.com/99designs/gqlgen/codegen/testserver/followschema.InnerDirectives`, tmp)
				return it, graphql.ErrorOnPath(ctx, err)
			}
		case "thirdParty":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("thirdParty"))
			directive0 := func(ctx context.Context) (any, error) {
				return ec.unmarshalOThirdParty2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐThirdParty(ctx, v)
			}

			directive1 := func(ctx context.Context) (any, error) {
				if ec.directives.Directive3 == nil {
					var zeroVal *ThirdParty
					return zeroVal, errors.New("directive directive3 is not implemented")
				}
				return ec.directives.Directive3(ctx, obj, directive0)
			}
			directive2 := func(ctx context.Context) (any, error) {
				min, err := ec.unmarshalNInt2int(ctx, 0)
				if err != nil {
					var zeroVal *ThirdParty
					return zeroVal, err
				}
				max, err := ec.unmarshalOInt2ᚖint(ctx, 7)
				if err != nil {
					var zeroVal *ThirdParty
					return zeroVal, err
				}
				if ec.directives.Length == nil {
					var zeroVal *ThirdParty
					return zeroVal, errors.New("directive length is not implemented")
				}
				return ec.directives.Length(ctx, obj, directive1, min, max, nil)
			}

			tmp, err := directive2(ctx)
			if err != nil {
				return it, graphql.ErrorOnPath(ctx, err)
			}
			if data, ok := tmp.(*ThirdParty); ok {
				it.ThirdParty = data
			} else if tmp == nil {
				it.ThirdParty = nil
			} else {
				err := fmt.Errorf(`unexpected type %T from directive, should be *github.com/99designs/gqlgen/codegen/testserver/followschema.ThirdParty`, tmp)
				return it, graphql.ErrorOnPath(ctx, err)
			}
		}
	}

	return it, nil
}

// endregion **************************** input.gotpl *****************************

// region    ************************** interface.gotpl ***************************

// endregion ************************** interface.gotpl ***************************

// region    **************************** object.gotpl ****************************

var objectDirectivesImplementors = []string{"ObjectDirectives"}

func (ec *executionContext) _ObjectDirectives(ctx context.Context, sel ast.SelectionSet, obj *ObjectDirectives) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, objectDirectivesImplementors)

	out := graphql.NewFieldSet(fields)
	deferred := make(map[string]*graphql.FieldSet)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("ObjectDirectives")
		case "text":
			out.Values[i] = ec._ObjectDirectives_text(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "nullableText":
			out.Values[i] = ec._ObjectDirectives_nullableText(ctx, field, obj)
		case "order":
			out.Values[i] = ec._ObjectDirectives_order(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx, ec.OperationContext)
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

var objectDirectivesWithCustomGoModelImplementors = []string{"ObjectDirectivesWithCustomGoModel"}

func (ec *executionContext) _ObjectDirectivesWithCustomGoModel(ctx context.Context, sel ast.SelectionSet, obj *ObjectDirectivesWithCustomGoModel) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, objectDirectivesWithCustomGoModelImplementors)

	out := graphql.NewFieldSet(fields)
	deferred := make(map[string]*graphql.FieldSet)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("ObjectDirectivesWithCustomGoModel")
		case "nullableText":
			out.Values[i] = ec._ObjectDirectivesWithCustomGoModel_nullableText(ctx, field, obj)
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx, ec.OperationContext)
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

func (ec *executionContext) unmarshalNInnerDirectives2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐInnerDirectives(ctx context.Context, v any) (*InnerDirectives, error) {
	res, err := ec.unmarshalInputInnerDirectives(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalNInputDirectives2githubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐInputDirectives(ctx context.Context, v any) (InputDirectives, error) {
	res, err := ec.unmarshalInputInputDirectives(ctx, v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOInnerDirectives2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐInnerDirectives(ctx context.Context, v any) (*InnerDirectives, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputInnerDirectives(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOInputDirectives2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐInputDirectives(ctx context.Context, v any) (*InputDirectives, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputInputDirectives(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOObjectDirectives2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐObjectDirectives(ctx context.Context, sel ast.SelectionSet, v *ObjectDirectives) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._ObjectDirectives(ctx, sel, v)
}

func (ec *executionContext) marshalOObjectDirectivesWithCustomGoModel2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋcodegenᚋtestserverᚋfollowschemaᚐObjectDirectivesWithCustomGoModel(ctx context.Context, sel ast.SelectionSet, v *ObjectDirectivesWithCustomGoModel) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._ObjectDirectivesWithCustomGoModel(ctx, sel, v)
}

// endregion ***************************** type.gotpl *****************************
