package codegen

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func buildObjects(types NamedTypes, s *schema.Schema, prog *loader.Program, imports Imports) Objects {
	var objects Objects

	for _, typ := range s.Types {
		switch typ := typ.(type) {
		case *schema.Object:
			obj := buildObject(types, typ, s)

			def, err := findGoType(prog, obj.Package, obj.GoType)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}
			if def != nil {
				bindObject(def.Type(), obj, imports)
			}

			objects = append(objects, obj)
		}
	}

	sort.Slice(objects, func(i, j int) bool {
		return strings.Compare(objects[i].GQLType, objects[j].GQLType) == -1
	})

	return objects
}

func buildObject(types NamedTypes, typ *schema.Object, s *schema.Schema) *Object {
	obj := &Object{NamedType: types[typ.TypeName()]}

	for _, i := range typ.Interfaces {
		obj.Satisfies = append(obj.Satisfies, i.Name)
	}

	for _, field := range typ.Fields {
		var args []FieldArgument
		for _, arg := range field.Args {
			newArg := FieldArgument{
				GQLName: arg.Name.Name,
				Type:    types.getType(arg.Type),
				Object:  obj,
			}

			if arg.Default != nil {
				newArg.Default = arg.Default.Value(nil)
				newArg.StripPtr()
			}
			args = append(args, newArg)
		}

		obj.Fields = append(obj.Fields, Field{
			GQLName: field.Name,
			Type:    types.getType(field.Type),
			Args:    args,
			Object:  obj,
		})
	}

	for name, typ := range s.EntryPoints {
		schemaObj := typ.(*schema.Object)
		if schemaObj.TypeName() != obj.GQLType {
			continue
		}

		obj.Root = true
		if name == "mutation" {
			obj.DisableConcurrency = true
		}
		if name == "subscription" {
			obj.Stream = true
		}
	}
	return obj
}
