package codegen

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func buildObjects(types NamedTypes, s *schema.Schema, prog *loader.Program) Objects {
	var objects Objects

	for _, typ := range s.Types {
		switch typ := typ.(type) {
		case *schema.Object:
			obj := buildObject(types, typ)

			def, err := findGoType(prog, obj.Package, obj.GoType)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}
			if def != nil {
				bindObject(def.Type(), obj)
			}

			objects = append(objects, obj)
		}
	}

	for name, typ := range s.EntryPoints {
		obj := typ.(*schema.Object)
		objects.ByName(obj.Name).Root = true
		if name == "mutation" {
			objects.ByName(obj.Name).DisableConcurrency = true
		}
		if name == "subscription" {
			objects.ByName(obj.Name).Stream = true
		}
	}

	sort.Slice(objects, func(i, j int) bool {
		return strings.Compare(objects[i].GQLType, objects[j].GQLType) == -1
	})

	return objects
}

func buildObject(types NamedTypes, typ *schema.Object) *Object {
	obj := &Object{NamedType: types[typ.TypeName()]}

	for _, i := range typ.Interfaces {
		obj.Satisfies = append(obj.Satisfies, i.Name)
	}

	for _, field := range typ.Fields {
		var args []FieldArgument
		for _, arg := range field.Args {
			args = append(args, FieldArgument{
				GQLName: arg.Name.Name,
				Type:    types.getType(arg.Type),
			})
		}

		obj.Fields = append(obj.Fields, Field{
			GQLName: field.Name,
			Type:    types.getType(field.Type),
			Args:    args,
			Object:  obj,
		})
	}
	return obj
}
