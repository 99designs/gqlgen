package codegen

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/vektah/gqlgen/neelance/common"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

// namedTypeFromSchema objects for every graphql type, including scalars. There should only be one instance of Type for each thing
func buildNamedTypes(s *schema.Schema, userTypes map[string]string) NamedTypes {
	types := map[string]*NamedType{}
	for _, schemaType := range s.Types {
		t := namedTypeFromSchema(schemaType)

		userType := userTypes[t.GQLType]
		if userType == "" {
			if t.IsScalar {
				userType = "github.com/vektah/gqlgen/graphql.String"
			} else {
				userType = "interface{}"
			}
		}
		t.Package, t.GoType = pkgAndType(userType)

		types[t.GQLType] = t
	}
	return types
}

func bindTypes(imports Imports, namedTypes NamedTypes, prog *loader.Program) {
	for _, t := range namedTypes {
		if t.Package == "" {
			continue
		}

		def, _ := findGoType(prog, t.Package, "Marshal"+t.GoType)
		switch def := def.(type) {
		case *types.Func:
			sig := def.Type().(*types.Signature)
			cpy := t.Ref
			t.Marshaler = &cpy

			t.Package, t.GoType = pkgAndType(sig.Params().At(0).Type().String())
			t.Import = imports.findByName(t.Package)
		}
	}
}

// namedTypeFromSchema objects for every graphql type, including primitives.
// don't recurse into object fields or interfaces yet, lets make sure we have collected everything first.
func namedTypeFromSchema(schemaType schema.NamedType) *NamedType {
	switch val := schemaType.(type) {
	case *schema.Scalar, *schema.Enum:
		return &NamedType{GQLType: val.TypeName(), IsScalar: true}
	case *schema.Interface, *schema.Union:
		return &NamedType{GQLType: val.TypeName(), IsInterface: true}
	default:
		return &NamedType{GQLType: val.TypeName()}
	}
}

// take a string in the form github.com/package/blah.Type and split it into package and type
func pkgAndType(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", name
	}

	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}

func (n NamedTypes) getType(t common.Type) *Type {
	var modifiers []string
	usePtr := true
	for {
		if _, nonNull := t.(*common.NonNull); nonNull {
			usePtr = false
		} else if _, nonNull := t.(*common.List); nonNull {
			usePtr = false
		} else {
			if usePtr {
				modifiers = append(modifiers, modPtr)
			}
			usePtr = true
		}

		switch val := t.(type) {
		case *common.NonNull:
			t = val.OfType
		case *common.List:
			modifiers = append(modifiers, modList)
			t = val.OfType
		case schema.NamedType:
			t := &Type{
				NamedType: n[val.TypeName()],
				Modifiers: modifiers,
			}

			if t.IsInterface && t.Modifiers[len(t.Modifiers)-1] == modPtr {
				t.Modifiers = t.Modifiers[0 : len(t.Modifiers)-1]
			}

			return t
		default:
			panic(fmt.Errorf("unknown type %T", t))
		}
	}
}
