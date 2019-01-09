package unified

import (
	"fmt"
	"go/build"
	"go/types"
	"os"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

func (g *Schema) FindGoType(pkgName string, typeName string) (types.Object, error) {
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

	pkg := g.Program.Imported[pkgName]
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

func findGoNamedType(def types.Type) (*types.Named, error) {
	if def == nil {
		return nil, nil
	}

	namedType, ok := def.(*types.Named)
	if !ok {
		return nil, errors.Errorf("expected %s to be a named type, instead found %T\n", def.String(), def)
	}

	return namedType, nil
}

func findGoInterface(def types.Type) (*types.Interface, error) {
	if def == nil {
		return nil, nil
	}
	namedType, err := findGoNamedType(def)
	if err != nil {
		return nil, err
	}
	if namedType == nil {
		return nil, nil
	}

	underlying, ok := namedType.Underlying().(*types.Interface)
	if !ok {
		return nil, errors.Errorf("expected %s to be a named interface, instead found %s", def.String(), namedType.String())
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

func equalFieldName(source, target string) bool {
	source = strings.Replace(source, "_", "", -1)
	target = strings.Replace(target, "_", "", -1)
	return strings.EqualFold(source, target)
}

// findField attempts to match the name to a struct field with the following
// priorites:
// 1. If struct tag is passed then struct tag has highest priority
// 2. Field in an embedded struct
// 3. Actual Field name
func findField(typ *types.Struct, name, structTag string) (*types.Var, error) {
	var foundField *types.Var
	foundFieldWasTag := false

	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)

		if structTag != "" {
			tags := reflect.StructTag(typ.Tag(i))
			if val, ok := tags.Lookup(structTag); ok {
				if equalFieldName(val, name) {
					if foundField != nil && foundFieldWasTag {
						return nil, errors.Errorf("tag %s is ambigious; multiple fields have the same tag value of %s", structTag, val)
					}

					foundField = field
					foundFieldWasTag = true
				}
			}
		}

		if field.Anonymous() {

			fieldType := field.Type()

			if ptr, ok := fieldType.(*types.Pointer); ok {
				fieldType = ptr.Elem()
			}

			// Type.Underlying() returns itself for all types except types.Named, where it returns a struct type.
			// It should be safe to always call.
			if named, ok := fieldType.Underlying().(*types.Struct); ok {
				f, err := findField(named, name, structTag)
				if err != nil && !strings.HasPrefix(err.Error(), "no field named") {
					return nil, err
				}
				if f != nil && foundField == nil {
					foundField = f
				}
			}
		}

		if !field.Exported() {
			continue
		}

		if equalFieldName(field.Name(), name) && foundField == nil { // aqui!
			foundField = field
		}
	}

	if foundField == nil {
		return nil, fmt.Errorf("no field named %s", name)
	}

	return foundField, nil
}

func resolvePkg(pkgName string) (string, error) {
	cwd, _ := os.Getwd()

	pkg, err := build.Default.Import(pkgName, cwd, build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.ImportPath, nil
}

var keywords = []string{
	"break",
	"default",
	"func",
	"interface",
	"select",
	"case",
	"defer",
	"go",
	"map",
	"struct",
	"chan",
	"else",
	"goto",
	"package",
	"switch",
	"const",
	"fallthrough",
	"if",
	"range",
	"type",
	"continue",
	"for",
	"import",
	"return",
	"var",
}

// sanitizeArgName prevents collisions with go keywords for arguments to resolver functions
func sanitizeArgName(name string) string {
	for _, k := range keywords {
		if name == k {
			return name + "Arg"
		}
	}
	return name
}
