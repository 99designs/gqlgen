package unified

import (
	"fmt"
	"go/types"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type BindError struct {
	object    *Object
	field     *Field
	typ       types.Type
	methodErr error
	varErr    error
}

func (b BindError) Error() string {
	return fmt.Sprintf(
		"\nunable to bind %s.%s to %s\n  %s\n  %s",
		b.object.Definition.GQLDefinition.Name,
		b.field.GQLName,
		b.typ.String(),
		b.methodErr.Error(),
		b.varErr.Error(),
	)
}

type BindErrors []BindError

func (b BindErrors) Error() string {
	var errs []string
	for _, err := range b {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, "\n\n")
}

func bindObject(object *Object, structTag string) BindErrors {
	var errs BindErrors
	for _, field := range object.Fields {
		if field.IsResolver {
			continue
		}

		// first try binding to a method
		methodErr := bindMethod(object.Definition.GoType, field)
		if methodErr == nil {
			continue
		}

		// otherwise try binding to a var
		varErr := bindVar(object.Definition.GoType, field, structTag)

		if varErr != nil {
			field.IsResolver = true

			errs = append(errs, BindError{
				object:    object,
				typ:       object.Definition.GoType,
				field:     field,
				varErr:    varErr,
				methodErr: methodErr,
			})
		}
	}
	return errs
}

func bindMethod(t types.Type, field *Field) error {
	namedType, ok := t.(*types.Named)
	if !ok {
		return fmt.Errorf("not a named type")
	}

	goName := field.GQLName
	if field.GoFieldName != "" {
		goName = field.GoFieldName
	}
	method := findMethod(namedType, goName)
	if method == nil {
		return fmt.Errorf("no method named %s", field.GQLName)
	}
	sig := method.Type().(*types.Signature)

	if sig.Results().Len() == 1 {
		field.NoErr = true
	} else if sig.Results().Len() != 2 {
		return fmt.Errorf("method has wrong number of args")
	}
	params := sig.Params()
	// If the first argument is the context, remove it from the comparison and set
	// the MethodHasContext flag so that the context will be passed to this model's method
	if params.Len() > 0 && params.At(0).Type().String() == "context.Context" {
		field.MethodHasContext = true
		vars := make([]*types.Var, params.Len()-1)
		for i := 1; i < params.Len(); i++ {
			vars[i-1] = params.At(i)
		}
		params = types.NewTuple(vars...)
	}

	newArgs, err := matchArgs(field, params)
	if err != nil {
		return err
	}

	result := sig.Results().At(0)
	if err := validateTypeBinding(field, result.Type()); err != nil {
		return errors.Wrap(err, "method has wrong return type")
	}

	// success, args and return type match. Bind to method
	field.GoFieldType = GoFieldMethod
	field.GoReceiverName = "obj"
	field.GoFieldName = method.Name()
	field.Args = newArgs
	return nil
}

func bindVar(t types.Type, field *Field, structTag string) error {
	underlying, ok := t.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("not a struct")
	}

	goName := field.GQLName
	if field.GoFieldName != "" {
		goName = field.GoFieldName
	}
	structField, err := findField(underlying, goName, structTag)
	if err != nil {
		return err
	}

	if err := validateTypeBinding(field, structField.Type()); err != nil {
		return errors.Wrap(err, "field has wrong type")
	}

	// success, bind to var
	field.GoFieldType = GoFieldVariable
	field.GoReceiverName = "obj"
	field.GoFieldName = structField.Name()
	return nil
}

func matchArgs(field *Field, params *types.Tuple) ([]FieldArgument, error) {
	var newArgs []FieldArgument

nextArg:
	for j := 0; j < params.Len(); j++ {
		param := params.At(j)
		for _, oldArg := range field.Args {
			if strings.EqualFold(oldArg.GQLName, param.Name()) {
				if !field.IsResolver {
					oldArg.TypeReference.GoType = param.Type()
				}
				newArgs = append(newArgs, oldArg)
				continue nextArg
			}
		}

		// no matching arg found, abort
		return nil, fmt.Errorf("arg %s not found on method", param.Name())
	}
	return newArgs, nil
}

func validateTypeBinding(field *Field, goType types.Type) error {
	gqlType := normalizeVendor(field.TypeReference.GoType.String())
	goTypeStr := normalizeVendor(goType.String())

	if equalTypes(goTypeStr, gqlType) {
		field.TypeReference.GoType = goType
		return nil
	}

	return fmt.Errorf("%s is not compatible with %s", gqlType, goTypeStr)
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

func normalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}

func equalTypes(goType string, gqlType string) bool {
	return goType == gqlType || "*"+goType == gqlType || goType == "*"+gqlType || strings.Replace(goType, "[]*", "[]", -1) == gqlType
}
