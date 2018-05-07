package codegen

import (
	"go/types"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

func findGoType(prog *loader.Program, pkgName string, typeName string) (types.Object, error) {
	if pkgName == "" {
		return nil, nil
	}
	fullName := typeName
	if pkgName != "" {
		fullName = pkgName + "." + typeName
	}

	pkgName, err := resolvePkg(pkgName)
	if err != nil {
		return nil, errors.Errorf("unable to resolve package for %s: %s\n", fullName, err.Error())
	}

	pkg := prog.Imported[pkgName]
	if pkg == nil {
		return nil, errors.Errorf("required package was not loaded: %s", fullName)
	}

	for astNode, def := range pkg.Defs {
		if astNode.Name != typeName || def.Parent() == nil || def.Parent() != pkg.Pkg.Scope() {
			continue
		}

		return def, nil
	}

	return nil, errors.Errorf("unable to find type %s\n", fullName)
}

func findGoNamedType(prog *loader.Program, pkgName string, typeName string) (*types.Named, error) {
	def, err := findGoType(prog, pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, nil
	}

	namedType, ok := def.Type().(*types.Named)
	if !ok {
		return nil, errors.Errorf("expected %s to be a named type, instead found %T\n", typeName, def.Type())
	}

	return namedType, nil
}

func findGoInterface(prog *loader.Program, pkgName string, typeName string) (*types.Interface, error) {
	namedType, err := findGoNamedType(prog, pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if namedType == nil {
		return nil, nil
	}

	underlying, ok := namedType.Underlying().(*types.Interface)
	if !ok {
		return nil, errors.Errorf("expected %s to be a named interface, instead found %s", typeName, namedType.String())
	}

	return underlying, nil
}

func findMethod(typ *types.Named, name string) *types.Func {
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
				if f := findMethod(named, name); f != nil {
					return f
				}
			}
		}
	}

	return nil
}

func findField(typ *types.Struct, name string) *types.Var {
	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		if field.Anonymous() {
			if named, ok := field.Type().(*types.Struct); ok {
				if f := findField(named, name); f != nil {
					return f
				}
			}

			if named, ok := field.Type().Underlying().(*types.Struct); ok {
				if f := findField(named, name); f != nil {
					return f
				}
			}
		}

		if !field.Exported() {
			continue
		}

		if strings.EqualFold(field.Name(), name) {
			return field
		}
	}
	return nil
}

func bindObject(t types.Type, object *Object, imports Imports) error {
	namedType, ok := t.(*types.Named)
	if !ok {
		return errors.Errorf("expected %s to be a named struct, instead found %s", object.FullName(), t.String())
	}

	underlying, ok := t.Underlying().(*types.Struct)
	if !ok {
		return errors.Errorf("expected %s to be a named struct, instead found %s", object.FullName(), t.String())
	}

	for i := range object.Fields {
		field := &object.Fields[i]
		if method := findMethod(namedType, field.GQLName); method != nil {
			sig := method.Type().(*types.Signature)
			field.GoMethodName = "obj." + method.Name()
			field.Type.Modifiers = modifiersFromGoType(sig.Results().At(0).Type())

			// check arg order matches code, not gql
			var newArgs []FieldArgument
		l2:
			for j := 0; j < sig.Params().Len(); j++ {
				param := sig.Params().At(j)
				for _, oldArg := range field.Args {
					if strings.EqualFold(oldArg.GQLName, param.Name()) {
						oldArg.Type.Modifiers = modifiersFromGoType(param.Type())
						newArgs = append(newArgs, oldArg)
						continue l2
					}
				}
				return errors.Errorf("cannot match argument " + param.Name() + " to any argument in " + t.String())
			}
			field.Args = newArgs

			if sig.Results().Len() == 1 {
				field.NoErr = true
			} else if sig.Results().Len() != 2 {
				return errors.Errorf("weird number of results on %s. expected either (result), or (result, error)\n", method.Name())
			}
			continue
		}

		if structField := findField(underlying, field.GQLName); structField != nil {
			field.Type.Modifiers = modifiersFromGoType(structField.Type())
			field.GoVarName = structField.Name()

			switch normalizeVendor(field.Type.FullSignature()) {
			case normalizeVendor(structField.Type().String()):
				// everything is fine

			case normalizeVendor(structField.Type().Underlying().String()):
				pkg, typ := pkgAndType(structField.Type().String())
				imp := imports.findByPkg(pkg)
				field.CastType = typ
				if imp.Name != "" {
					field.CastType = imp.Name + "." + typ
				}

			default:
				return errors.Errorf("type mismatch on %s.%s, expected %s got %s\n", object.GQLType, field.GQLName, field.Type.FullSignature(), structField.Type())
			}
			continue
		}
	}
	return nil
}

func modifiersFromGoType(t types.Type) []string {
	var modifiers []string
	for {
		switch val := t.(type) {
		case *types.Pointer:
			modifiers = append(modifiers, modPtr)
			t = val.Elem()
		case *types.Array:
			modifiers = append(modifiers, modList)
			t = val.Elem()
		case *types.Slice:
			modifiers = append(modifiers, modList)
			t = val.Elem()
		default:
			return modifiers
		}
	}
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

func normalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}
