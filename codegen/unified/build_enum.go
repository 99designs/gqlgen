package unified

import (
	"go/types"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

func (g *Schema) buildEnum(typ *ast.Definition) *Enum {
	namedType := g.NamedTypes[typ.Name]
	if typ.Kind != ast.Enum || strings.HasPrefix(typ.Name, "__") || g.Config.Models.UserDefined(typ.Name) {
		return nil
	}

	var values []EnumValue
	for _, v := range typ.EnumValues {
		values = append(values, EnumValue{v.Name, v.Description})
	}

	enum := Enum{
		Definition: namedType,
		Values:     values,
		InTypemap:  g.Config.Models.UserDefined(typ.Name),
	}

	enum.Definition.GoType = types.NewNamed(types.NewTypeName(0, g.Config.Model.Pkg(), templates.ToCamel(enum.Definition.GQLDefinition.Name), nil), nil, nil)

	return &enum
}
