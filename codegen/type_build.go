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
func (cfg *Config) buildNamedTypes() NamedTypes {
	types := map[string]*NamedType{}
	for _, schemaType := range cfg.schema.Types {
		t := namedTypeFromSchema(schemaType)

		userType := cfg.Typemap[t.GQLType]
		t.IsUserDefined = userType != ""
		if userType == "" && t.IsScalar {
			userType = "github.com/vektah/gqlgen/graphql.String"
		}

		if userType != "" {
			t.Package, t.GoType = pkgAndType(userType)
		}

		types[t.GQLType] = t
	}
	return types
}

func (cfg *Config) bindTypes(imports Imports, namedTypes NamedTypes, destDir string, prog *loader.Program) Imports {
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
			imports, t.Import = imports.addPkg(namedTypes, destDir, t.Package)
		}
	}
	return imports
}

// namedTypeFromSchema objects for every graphql type, including primitives.
// don't recurse into object fields or interfaces yet, lets make sure we have collected everything first.
func namedTypeFromSchema(schemaType schema.NamedType) *NamedType {
	switch val := schemaType.(type) {
	case *schema.Scalar, *schema.Enum:
		return &NamedType{GQLType: val.TypeName(), IsScalar: true}
	case *schema.Interface, *schema.Union:
		return &NamedType{GQLType: val.TypeName(), IsInterface: true}
	case *schema.InputObject:
		return &NamedType{GQLType: val.TypeName(), IsInput: true}
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

	return normalizeVendor(strings.Join(parts[:len(parts)-1], ".")), parts[len(parts)-1]
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

			if t.IsInterface {
				t.StripPtr()
			}

			return t
		default:
			panic(fmt.Errorf("unknown type %T", t))
		}
	}
}
