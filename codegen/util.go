package codegen

import (
	"fmt"
	"go/types"
	"os"
	"strings"

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
		return nil, fmt.Errorf("unable to resolve package for %s: %s\n", fullName, err.Error())
	}

	pkg := prog.Imported[pkgName]
	if pkg == nil {
		return nil, fmt.Errorf("required package was not loaded: %s", fullName)
	}

	for astNode, def := range pkg.Defs {
		if astNode.Name != typeName || isMethod(def) {
			continue
		}

		return def, nil
	}
	return nil, fmt.Errorf("unable to find type %s\n", fullName)
}

func isMethod(t types.Object) bool {
	f, isFunc := t.(*types.Func)
	if !isFunc {
		return false
	}

	return f.Type().(*types.Signature).Recv() != nil
}

func bindObject(t types.Type, object *Object) bool {
	switch t := t.(type) {
	case *types.Named:
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			if !method.Exported() {
				continue
			}

			if methodField := object.GetField(method.Name()); methodField != nil {
				methodField.GoMethodName = "it." + method.Name()
				sig := method.Type().(*types.Signature)

				methodField.Type.Modifiers = modifiersFromGoType(sig.Results().At(0).Type())

				// check arg order matches code, not gql

				var newArgs []FieldArgument
			l2:
				for j := 0; j < sig.Params().Len(); j++ {
					param := sig.Params().At(j)
					for _, oldArg := range methodField.Args {
						if strings.EqualFold(oldArg.GQLName, param.Name()) {
							oldArg.Type.Modifiers = modifiersFromGoType(param.Type())
							newArgs = append(newArgs, oldArg)
							continue l2
						}
					}
					fmt.Fprintln(os.Stderr, "cannot match argument "+param.Name()+" to any argument in "+t.String())
				}
				methodField.Args = newArgs

				if sig.Results().Len() == 1 {
					methodField.NoErr = true
				} else if sig.Results().Len() != 2 {
					fmt.Fprintf(os.Stderr, "weird number of results on %s. expected either (result), or (result, error)\n", method.Name())
				}
			}
		}

		bindObject(t.Underlying(), object)
		return true

	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			// Todo: struct tags, name and - at least

			if !field.Exported() {
				continue
			}

			// Todo: check for type matches before binding too?
			if objectField := object.GetField(field.Name()); objectField != nil {
				objectField.GoVarName = "it." + field.Name()
				objectField.Type.Modifiers = modifiersFromGoType(field.Type())
			}
		}
		t.Underlying()
		return true
	}

	return false
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
