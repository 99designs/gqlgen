package codegen

import (
	"fmt"
	"go/types"
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func (b *builder) bindMethod(t types.Type, field *Field) error {
	namedType, err := findGoNamedType(t)
	if err != nil {
		return err
	}

	method := b.findMethod(namedType, field.GoFieldName)
	if method == nil {
		return fmt.Errorf("no method named %s", field.GoFieldName)
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

	if err := b.bindArgs(field, params); err != nil {
		return err
	}

	result := sig.Results().At(0)
	if err := compatibleTypes(field.TypeReference.GoType, result.Type()); err != nil {
		return errors.Wrapf(err, "%s is not compatible with %s", field.TypeReference.GoType.String(), result.String())
	}

	// success, args and return type match. Bind to method
	field.GoFieldType = GoFieldMethod
	field.GoReceiverName = "obj"
	field.GoFieldName = method.Name()
	field.TypeReference.GoType = result.Type()
	return nil
}

func (b *builder) bindVar(t types.Type, field *Field) error {
	underlying, ok := t.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("not a struct")
	}

	structField, err := b.findField(underlying, field.GoFieldName)
	if err != nil {
		return err
	}

	if err := compatibleTypes(field.TypeReference.GoType, structField.Type()); err != nil {
		return errors.Wrapf(err, "%s is not compatible with %s", field.TypeReference.GoType.String(), field.TypeReference.GoType.String())
	}

	// success, bind to var
	field.GoFieldType = GoFieldVariable
	field.GoReceiverName = "obj"
	field.GoFieldName = structField.Name()
	field.TypeReference.GoType = structField.Type()
	return nil
}

func (b *builder) bindArgs(field *Field, params *types.Tuple) error {
	var newArgs []*FieldArgument

nextArg:
	for j := 0; j < params.Len(); j++ {
		param := params.At(j)
		for _, oldArg := range field.Args {
			if strings.EqualFold(oldArg.GQLName, param.Name()) {
				oldArg.TypeReference.GoType = param.Type()
				newArgs = append(newArgs, oldArg)
				continue nextArg
			}
		}

		// no matching arg found, abort
		return fmt.Errorf("arg %s not found on method", param.Name())
	}

	field.Args = newArgs
	return nil
}

// compatibleTypes isnt a strict comparison, it allows for pointer differences
func compatibleTypes(expected types.Type, actual types.Type) error {
	//fmt.Println("Comparing ", expected.String(), actual.String())

	// Special case to deal with pointer mismatches
	{
		expectedPtr, expectedIsPtr := expected.(*types.Pointer)
		actualPtr, actualIsPtr := actual.(*types.Pointer)

		if expectedIsPtr && actualIsPtr {
			return compatibleTypes(expectedPtr.Elem(), actualPtr.Elem())
		}
		if expectedIsPtr && !actualIsPtr {
			return compatibleTypes(expectedPtr.Elem(), actual)
		}
		if !expectedIsPtr && actualIsPtr {
			return compatibleTypes(expected, actualPtr.Elem())
		}
	}

	switch expected := expected.(type) {
	case *types.Slice:
		if actual, ok := actual.(*types.Slice); ok {
			return compatibleTypes(expected.Elem(), actual.Elem())
		}

	case *types.Array:
		if actual, ok := actual.(*types.Array); ok {
			if expected.Len() != actual.Len() {
				return fmt.Errorf("array length differs")
			}

			return compatibleTypes(expected.Elem(), actual.Elem())
		}

	case *types.Basic:
		if actual, ok := actual.(*types.Basic); ok {
			if actual.Kind() != expected.Kind() {
				return fmt.Errorf("basic kind differs, %s != %s", expected.Name(), actual.Name())
			}

			return nil
		}

	case *types.Struct:
		if actual, ok := actual.(*types.Struct); ok {
			if expected.NumFields() != actual.NumFields() {
				return fmt.Errorf("number of struct fields differ")
			}

			for i := 0; i < expected.NumFields(); i++ {
				if expected.Field(i).Name() != actual.Field(i).Name() {
					return fmt.Errorf("struct field %d name differs, %s != %s", i, expected.Field(i).Name(), actual.Field(i).Name())
				}
				if err := compatibleTypes(expected.Field(i).Type(), actual.Field(i).Type()); err != nil {
					return err
				}
			}
			return nil
		}

	case *types.Tuple:
		if actual, ok := actual.(*types.Tuple); ok {
			if expected.Len() != actual.Len() {
				return fmt.Errorf("tuple length differs, %d != %d", expected.Len(), actual.Len())
			}

			for i := 0; i < expected.Len(); i++ {
				if err := compatibleTypes(expected.At(i).Type(), actual.At(i).Type()); err != nil {
					return err
				}
			}

			return nil
		}

	case *types.Signature:
		if actual, ok := actual.(*types.Signature); ok {
			if err := compatibleTypes(expected.Params(), actual.Params()); err != nil {
				return err
			}
			if err := compatibleTypes(expected.Results(), actual.Results()); err != nil {
				return err
			}

			return nil
		}
	case *types.Interface:
		if actual, ok := actual.(*types.Interface); ok {
			if expected.NumMethods() != actual.NumMethods() {
				return fmt.Errorf("interface method count differs, %d != %d", expected.NumMethods(), actual.NumMethods())
			}

			for i := 0; i < expected.NumMethods(); i++ {
				if expected.Method(i).Name() != actual.Method(i).Name() {
					return fmt.Errorf("interface method %d name differs, %s != %s", i, expected.Method(i).Name(), actual.Method(i).Name())
				}
				if err := compatibleTypes(expected.Method(i).Type(), actual.Method(i).Type()); err != nil {
					return err
				}
			}

			return nil
		}

	case *types.Map:
		if actual, ok := actual.(*types.Map); ok {
			if err := compatibleTypes(expected.Key(), actual.Key()); err != nil {
				return err
			}

			if err := compatibleTypes(expected.Elem(), actual.Elem()); err != nil {
				return err
			}

			return nil
		}

	case *types.Chan:
		if actual, ok := actual.(*types.Chan); ok {
			return compatibleTypes(expected.Elem(), actual.Elem())
		}

	case *types.Named:
		if actual, ok := actual.(*types.Named); ok {
			if normalizeVendor(expected.Obj().Pkg().Path()) != normalizeVendor(actual.Obj().Pkg().Path()) {
				return fmt.Errorf(
					"package name of named type differs, %s != %s",
					normalizeVendor(expected.Obj().Pkg().Path()),
					normalizeVendor(actual.Obj().Pkg().Path()),
				)
			}

			if expected.Obj().Name() != actual.Obj().Name() {
				return fmt.Errorf(
					"named type name differs, %s != %s",
					normalizeVendor(expected.Obj().Name()),
					normalizeVendor(actual.Obj().Name()),
				)
			}

			return nil
		}

		// Before models are generated all missing references will be Invalid Basic references.
		// lets assume these are valid too.
		if actual, ok := actual.(*types.Basic); ok && actual.Kind() == types.Invalid {
			return nil
		}

	default:
		return fmt.Errorf("missing support for %T", expected)
	}

	return fmt.Errorf("type mismatch %T != %T", expected, actual)
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

func normalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}

func (b *builder) findMethod(typ *types.Named, name string) *types.Func {
	for i := 0; i < typ.NumMethods(); i++ {
		method := typ.Method(i)
		if !method.Exported() {
			continue
		}

		if strings.EqualFold(method.Name(), name) {
			return method
		}
	}

	if s, ok := typ.Underlying().(*types.Struct); ok {
		for i := 0; i < s.NumFields(); i++ {
			field := s.Field(i)
			if !field.Anonymous() {
				continue
			}

			if named, ok := field.Type().(*types.Named); ok {
				if f := b.findMethod(named, name); f != nil {
					return f
				}
			}
		}
	}

	return nil
}

// findField attempts to match the name to a struct field with the following
// priorites:
// 1. If struct tag is passed then struct tag has highest priority
// 2. Actual Field name
// 3. Field in an embedded struct
func (b *builder) findField(typ *types.Struct, name string) (*types.Var, error) {
	if b.Config.StructTag != "" {
		var foundField *types.Var
		for i := 0; i < typ.NumFields(); i++ {
			field := typ.Field(i)
			if !field.Exported() {
				continue
			}
			tags := reflect.StructTag(typ.Tag(i))
			if val, ok := tags.Lookup(b.Config.StructTag); ok && equalFieldName(val, name) {
				if foundField != nil {
					return nil, errors.Errorf("tag %s is ambigious; multiple fields have the same tag value of %s", b.Config.StructTag, val)
				}

				foundField = field
			}
		}
		if foundField != nil {
			return foundField, nil
		}
	}

	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		if !field.Exported() {
			continue
		}
		if equalFieldName(field.Name(), name) { // aqui!
			return field, nil
		}
	}

	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		if !field.Exported() {
			continue
		}

		if field.Anonymous() {
			fieldType := field.Type()

			if ptr, ok := fieldType.(*types.Pointer); ok {
				fieldType = ptr.Elem()
			}

			// Type.Underlying() returns itself for all types except types.Named, where it returns a struct type.
			// It should be safe to always call.
			if named, ok := fieldType.Underlying().(*types.Struct); ok {
				f, err := b.findField(named, name)
				if err != nil && !strings.HasPrefix(err.Error(), "no field named") {
					return nil, err
				}
				if f != nil {
					return f, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no field named %s", name)
}
