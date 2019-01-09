package unified

import (
	"go/types"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

func (g *Schema) buildInput(typ *ast.Definition) (*Object, error) {
	obj := &Object{
		Definition: g.NamedTypes[typ.Name],
		InTypemap:  g.Config.Models.UserDefined(typ.Name),
	}

	for _, field := range typ.Fields {
		newField, err := g.buildField(obj, field)
		if err != nil {
			return nil, err
		}

		if !newField.TypeReference.Definition.GQLDefinition.IsInputType() {
			return nil, errors.Errorf(
				"%s cannot be used as a field of %s. only input and scalar types are allowed",
				newField.Definition.GQLDefinition.Name,
				obj.Definition.GQLDefinition.Name,
			)
		}

		obj.Fields = append(obj.Fields, newField)

	}
	dirs, err := g.getDirectives(typ.Directives)
	if err != nil {
		return nil, err
	}
	obj.Directives = dirs

	if _, isMap := obj.Definition.GoType.(*types.Map); !isMap && obj.InTypemap {
		bindErrs := bindObject(obj, g.Config.StructTag)
		if len(bindErrs) > 0 {
			return nil, bindErrs
		}
	}

	return obj, nil
}
