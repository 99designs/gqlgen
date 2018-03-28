package codegen

import (
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
)

func buildEnums(types NamedTypes, s *schema.Schema) []Enum {
	var enums []Enum

	for _, typ := range s.Types {
		if strings.HasPrefix(typ.TypeName(), "__") {
			continue
		}
		if e, ok := typ.(*schema.Enum); ok {
			var values []EnumValue
			for _, v := range e.Values {
				values = append(values, EnumValue{v.Name, v.Desc})
			}

			enum := Enum{
				NamedType: types[e.TypeName()],
				Values:    values,
			}
			enum.GoType = ucFirst(enum.GQLType)
			enums = append(enums, enum)
		}
	}

	sort.Slice(enums, func(i, j int) bool {
		return strings.Compare(enums[i].GQLType, enums[j].GQLType) == -1
	})

	return enums
}
