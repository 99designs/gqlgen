package codegen

import (
	"go/types"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

func (g *Generator) buildEnums(ts NamedTypes) []Enum {
	var enums []Enum

	for _, typ := range g.schema.Types {
		namedType := ts[typ.Name]
		if typ.Kind != ast.Enum || strings.HasPrefix(typ.Name, "__") || g.Models.UserDefined(typ.Name) {
			continue
		}

		var values []EnumValue
		for _, v := range typ.EnumValues {
			values = append(values, EnumValue{v.Name, v.Description})
		}

		enum := Enum{
			Definition:  namedType,
			Values:      values,
			Description: typ.Description,
		}

		enum.Definition.GoType = types.NewNamed(types.NewTypeName(0, g.Config.Model.Pkg(), templates.ToCamel(enum.Definition.GQLType), nil), nil, nil)

		enums = append(enums, enum)
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Definition.GQLType < enums[j].Definition.GQLType
	})

	return enums
}
