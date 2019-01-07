package codegen

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

// namedTypeFromSchema objects for every graphql type, including scalars. There should only be one instance of TypeReference for each thing
func (g *Generator) buildNamedTypes(prog *loader.Program) (NamedTypes, error) {
	ts := map[string]*TypeDefinition{}
	for _, schemaType := range g.schema.Types {
		t := namedTypeFromSchema(schemaType)
		ts[t.GQLType] = t

		var pkgName, typeName string
		if userEntry, ok := g.Models[t.GQLType]; ok && userEntry.Model != "" {
			// special case for maps
			if userEntry.Model == "map[string]interface{}" {
				t.GoType = types.NewMap(types.Typ[types.String], types.NewInterface(nil, nil).Complete())

				continue
			}

			pkgName, typeName = pkgAndType(userEntry.Model)
		} else if t.IsScalar {
			pkgName = "github.com/99designs/gqlgen/graphql"
			typeName = "String"
		} else {
			// Missing models, but we need to set up the types so any references will point to the code that will
			// get generated
			t.GoType = types.NewNamed(types.NewTypeName(0, g.Config.Model.Pkg(), templates.ToCamel(t.GQLType), nil), nil, nil)

			continue
		}

		if pkgName == "" {
			return nil, fmt.Errorf("missing package name for %s", schemaType.Name)
		}

		// External marshal functions
		def, _ := findGoType(prog, pkgName, "Marshal"+typeName)
		if f, isFunc := def.(*types.Func); isFunc {
			sig := def.Type().(*types.Signature)
			t.GoType = sig.Params().At(0).Type()
			t.Marshaler = f

			unmarshal, err := findGoType(prog, pkgName, "Unmarshal"+typeName)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to find unmarshal func for %s.%s", pkgName, typeName)
			}
			t.Unmarshaler = unmarshal.(*types.Func)
			continue
		}

		// Normal object binding
		obj, err := findGoType(prog, pkgName, typeName)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find %s.%s", pkgName, typeName)
		}
		t.GoType = obj.Type()

		namedType := obj.Type().(*types.Named)
		hasUnmarshal := false
		for i := 0; i < namedType.NumMethods(); i++ {
			switch namedType.Method(i).Name() {
			case "UnmarshalGQL":
				hasUnmarshal = true
			}
		}

		// Special case to reference generated unmarshal functions
		if !hasUnmarshal {
			t.Unmarshaler = types.NewFunc(0, g.Config.Exec.Pkg(), "Unmarshal"+schemaType.Name, nil)
		}

	}
	return ts, nil
}

// namedTypeFromSchema objects for every graphql type, including primitives.
// don't recurse into object fields or interfaces yet, lets make sure we have collected everything first.
func namedTypeFromSchema(schemaType *ast.Definition) *TypeDefinition {
	switch schemaType.Kind {
	case ast.Scalar, ast.Enum:
		return &TypeDefinition{GQLType: schemaType.Name, IsScalar: true}
	case ast.Interface, ast.Union:
		return &TypeDefinition{GQLType: schemaType.Name, IsInterface: true}
	case ast.InputObject:
		return &TypeDefinition{GQLType: schemaType.Name, IsInput: true}
	default:
		return &TypeDefinition{GQLType: schemaType.Name}
	}
}

// take a string in the form github.com/package/blah.TypeReference and split it into package and type
func pkgAndType(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", name
	}

	return normalizeVendor(strings.Join(parts[:len(parts)-1], ".")), parts[len(parts)-1]
}
