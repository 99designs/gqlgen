package unified

import (
	"go/types"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

func (b *builder) buildEnum(typ *ast.Definition) *Enum {
	namedType := b.NamedTypes[typ.Name]
	if typ.Kind != ast.Enum || strings.HasPrefix(typ.Name, "__") || b.Config.Models.UserDefined(typ.Name) {
		return nil
	}

	var values []EnumValue
	for _, v := range typ.EnumValues {
		values = append(values, EnumValue{v.Name, v.Description})
	}

	enum := Enum{
		Definition: namedType,
		Values:     values,
		InTypemap:  b.Config.Models.UserDefined(typ.Name),
	}

	enum.Definition.GoType = types.NewNamed(types.NewTypeName(0, b.Config.Model.Pkg(), templates.ToCamel(enum.Definition.GQLDefinition.Name), nil), nil, nil)

	return &enum
}
