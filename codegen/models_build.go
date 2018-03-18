package codegen

import (
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func buildModels(types NamedTypes, s *schema.Schema, prog *loader.Program) []Model {
	var models []Model

	for _, typ := range s.Types {
		var model Model
		switch typ := typ.(type) {
		case *schema.Object:
			obj := buildObject(types, typ, s)
			if obj.Root || obj.GoType != "" {
				continue
			}
			model = obj2Model(s, obj)
		case *schema.InputObject:
			obj := buildInput(types, typ)
			if obj.GoType != "" {
				continue
			}
			model = obj2Model(s, obj)
		case *schema.Interface, *schema.Union:
			intf := buildInterface(types, typ, prog)
			if intf.GoType != "" {
				continue
			}
			model = int2Model(intf)
		default:
			continue
		}

		models = append(models, model)
	}

	sort.Slice(models, func(i, j int) bool {
		return strings.Compare(models[i].GQLType, models[j].GQLType) == -1
	})

	return models
}

func obj2Model(s *schema.Schema, obj *Object) Model {
	model := Model{
		NamedType: obj.NamedType,
		Fields:    []ModelField{},
	}

	model.GoType = ucFirst(obj.GQLType)
	model.Marshaler = &Ref{GoType: obj.GoType}

	for i := range obj.Fields {
		field := &obj.Fields[i]
		mf := ModelField{Type: field.Type}

		if mf.IsScalar {
			mf.GoVarName = ucFirst(field.GQLName)
			if mf.GoVarName == "Id" {
				mf.GoVarName = "ID"
			}
		} else if mf.IsInput {
			mf.GoVarName = ucFirst(field.GQLName)
		} else if mf.IsSlice() {
			// one to many, we don't need a prop, we need a resolver
		} else {
			mf.GoFKName = ucFirst(field.GQLName) + "ID"
			mf.GoFKType = "string"

			if obj, ok := s.Types[field.GQLType].(*schema.Object); ok {
				for _, f := range obj.Fields {
					if strings.EqualFold(f.Name, "id") {
						if strings.Contains(f.Type.String(), "Int") {
							mf.GoFKType = "int"
						}
					}
				}
			}
		}

		model.Fields = append(model.Fields, mf)
	}

	return model
}

func int2Model(obj *Interface) Model {
	model := Model{
		NamedType: obj.NamedType,
		Fields:    []ModelField{},
	}

	model.GoType = ucFirst(obj.GQLType)
	model.Marshaler = &Ref{GoType: obj.GoType}

	return model
}
