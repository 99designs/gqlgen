package codegen

import (
	"sort"

	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (g *Generator) buildModels(types NamedTypes, prog *loader.Program) ([]Model, error) {
	var models []Model

	for _, typ := range g.schema.Types {
		var model Model
		if g.Models.UserDefined(typ.Name) {
			continue
		}
		switch typ.Kind {
		case ast.Object:
			obj, err := g.buildObject(prog, types, typ)
			if err != nil {
				return nil, err
			}
			if obj.Root {
				continue
			}
			model = g.obj2Model(obj)
		case ast.InputObject:
			obj, err := g.buildInput(types, typ)
			if err != nil {
				return nil, err
			}
			model = g.obj2Model(obj)
		case ast.Interface, ast.Union:
			intf := g.buildInterface(types, typ, prog)
			model = int2Model(intf)
		default:
			continue
		}
		model.Description = typ.Description // It's this or change both obj2Model and buildObject

		models = append(models, model)
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Definition.GQLType < models[j].Definition.GQLType
	})

	return models, nil
}

func (g *Generator) obj2Model(obj *Object) Model {
	model := Model{
		Definition: obj.Definition,
		Implements: obj.Implements,
		Fields:     []ModelField{},
	}

	for i := range obj.Fields {
		field := &obj.Fields[i]
		mf := ModelField{TypeReference: field.TypeReference, GQLName: field.GQLName}

		if field.GoFieldName != "" {
			mf.GoFieldName = field.GoFieldName
		} else {
			mf.GoFieldName = field.GoNameExported()
		}

		model.Fields = append(model.Fields, mf)
	}

	return model
}

func int2Model(obj *Interface) Model {
	model := Model{
		Definition: obj.Definition,
		Fields:     []ModelField{},
	}

	return model
}
