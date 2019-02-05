package codegen

import (
	"fmt"
	"go/types"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/internal/code"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

func (b *builder) buildTypeDef(schemaType *ast.Definition) (*TypeDefinition, error) {
	t := &TypeDefinition{
		GQLDefinition: schemaType,
	}

	var pkgName, typeName string
	if userEntry, ok := b.Config.Models[t.GQLDefinition.Name]; ok && userEntry.Model != "" {
		// special case for maps
		if userEntry.Model == "map[string]interface{}" {
			t.GoType = config.MapType

			return t, nil
		}

		if userEntry.Model == "interface{}" {
			t.GoType = config.InterfaceType

			return t, nil
		}

		pkgName, typeName = code.PkgAndType(userEntry.Model)
		if pkgName == "" {
			return nil, fmt.Errorf("missing package name for %s", schemaType.Name)
		}

	} else if t.GQLDefinition.Kind == ast.Scalar {
		pkgName = "github.com/99designs/gqlgen/graphql"
		typeName = "String"
	} else {
		// Missing models, but we need to set up the types so any references will point to the code that will
		// get generated
		t.GoType = types.NewNamed(types.NewTypeName(0, b.Config.Model.Pkg(), templates.ToCamel(t.GQLDefinition.Name), nil), nil, nil)

		if t.GQLDefinition.Kind != ast.Enum {
			t.Unmarshaler = types.NewFunc(0, b.Config.Exec.Pkg(), "Unmarshal"+schemaType.Name, nil)
		}

		return t, nil
	}

	// External marshal functions
	def, err := b.Binder.FindObject(pkgName, typeName)
	if err != nil {
		return nil, err
	}
	if f, isFunc := def.(*types.Func); isFunc {
		sig := def.Type().(*types.Signature)
		t.GoType = sig.Params().At(0).Type()
		t.Marshaler = f

		unmarshal, err := b.Binder.FindObject(pkgName, "Unmarshal"+typeName)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find unmarshal func for %s.%s", pkgName, typeName)
		}
		t.Unmarshaler = unmarshal.(*types.Func)
		return t, nil
	}

	// Normal object binding
	t.GoType = def.Type()

	namedType := def.Type().(*types.Named)
	hasUnmarshal := false
	for i := 0; i < namedType.NumMethods(); i++ {
		switch namedType.Method(i).Name() {
		case "UnmarshalGQL":
			hasUnmarshal = true
		}
	}

	// Special case to reference generated unmarshal functions
	if !hasUnmarshal {
		t.Unmarshaler = types.NewFunc(0, b.Config.Exec.Pkg(), "unmarshalInput"+schemaType.Name, nil)
	}

	return t, nil
}

func (n NamedTypes) goTypeForAst(t *ast.Type) types.Type {
	if t.Elem != nil {
		return types.NewSlice(n.goTypeForAst(t.Elem))
	}

	nt := n[t.NamedType]
	gt := nt.GoType
	if gt == nil {
		panic("missing type " + t.NamedType)
	}

	if !t.NonNull && nt.GQLDefinition.Kind != ast.Interface {
		return types.NewPointer(gt)
	}

	return gt
}

func (n NamedTypes) getType(t *ast.Type) *TypeReference {
	return &TypeReference{
		Definition: n[t.Name()],
		GoType:     n.goTypeForAst(t),
		ASTType:    t,
	}
}
