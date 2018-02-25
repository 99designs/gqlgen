package codegen

import (
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
)

func buildModels(types NamedTypes, s *schema.Schema) Objects {
	var models Objects

	for _, typ := range s.Types {
		var model *Object
		switch typ := typ.(type) {
		case *schema.Object:
			model = buildObject(types, typ, s)

		case *schema.InputObject:
			model = buildInput(types, typ)
		}

		if model == nil || model.Root || model.GoType != "" {
			continue
		}

		bindGenerated(types, model)

		models = append(models, model)
	}

	sort.Slice(models, func(i, j int) bool {
		return strings.Compare(models[i].GQLType, models[j].GQLType) == -1
	})

	return models
}

func bindGenerated(types NamedTypes, object *Object) {
	object.GoType = ucFirst(object.GQLType)
	object.Marshaler = &Ref{GoType: object.GoType}

	for i := range object.Fields {
		field := &object.Fields[i]

		if field.IsScalar {
			field.GoVarName = ucFirst(field.GQLName)
			if field.GoVarName == "Id" {
				field.GoVarName = "ID"
			}
		} else if object.Input {
			field.GoFKName = ucFirst(field.GQLName)
			field.GoFKType = types[field.GQLType].GoType
		} else {
			field.GoFKName = ucFirst(field.GQLName) + "ID"
			field.GoFKType = "int" // todo: use schema to determine type of id?
		}
	}
}
