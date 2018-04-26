package codegen

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildObjects(types NamedTypes, prog *loader.Program, imports Imports) (Objects, error) {
	var objects Objects

	for _, typ := range cfg.schema.Types {
		switch typ := typ.(type) {
		case *schema.Object:
			obj, err := cfg.buildObject(types, typ)
			if err != nil {
				return nil, err
			}

			def, err := findGoType(prog, obj.Package, obj.GoType)
			if err != nil {
				return nil, err
			}
			if def != nil {
				err = bindObject(def.Type(), obj, imports)
				if err != nil {
					return nil, err
				}
			}

			objects = append(objects, obj)
		}
	}

	sort.Slice(objects, func(i, j int) bool {
		return strings.Compare(objects[i].GQLType, objects[j].GQLType) == -1
	})

	return objects, nil
}

func (cfg *Config) buildObject(types NamedTypes, typ *schema.Object) (*Object, error) {
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

			if !newArg.Type.IsInput && !newArg.Type.IsScalar {
				return nil, errors.Errorf("%s cannot be used as argument of %s.%s. only input and scalar types are allowed", arg.Type, obj.GQLType, field.Name)
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

	for name, typ := range cfg.schema.EntryPoints {
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
	return obj, nil
}
