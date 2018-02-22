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
			obj := buildObject(types, typ)

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

func findMissing(objects Objects) Objects {
	var missingObjects Objects

	for _, object := range objects {
		if !object.Generated || object.Root {
			continue
		}
		object.GoType = ucFirst(object.GQLType)
		object.Marshaler = &Ref{GoType: object.GoType}

		for i := range object.Fields {
			field := &object.Fields[i]

			if field.IsScalar {
				field.GoVarName = ucFirst(field.GQLName)
				if field.GoVarName == "Id" {
					field.GoVarName = "ID"
				}
			} else {
				field.GoFKName = ucFirst(field.GQLName) + "ID"
				field.GoFKType = "int"

				for _, f := range objects.ByName(field.Type.GQLType).Fields {
					if strings.EqualFold(f.GQLName, "id") {
						field.GoFKType = f.GoType
					}
				}
			}
		}

		missingObjects = append(missingObjects, object)
	}

	sort.Slice(missingObjects, func(i, j int) bool {
		return strings.Compare(missingObjects[i].GQLType, missingObjects[j].GQLType) == -1
	})

	return missingObjects
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
				Object:  obj,
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
