package codegen

import (
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
)

func buildInterfaces(types NamedTypes, s *schema.Schema) []*Interface {
	var interfaces []*Interface
	for _, typ := range s.Types {
		switch typ := typ.(type) {

		case *schema.Union:
			i := &Interface{NamedType: types[typ.TypeName()]}

			for _, implementor := range typ.PossibleTypes {
				i.Implementors = append(i.Implementors, types[implementor.TypeName()])
			}

			interfaces = append(interfaces, i)

		case *schema.Interface:
			i := &Interface{NamedType: types[typ.TypeName()]}

			for _, implementor := range typ.PossibleTypes {
				i.Implementors = append(i.Implementors, types[implementor.TypeName()])
			}

			interfaces = append(interfaces, i)
		}
	}

	sort.Slice(interfaces, func(i, j int) bool {
		return strings.Compare(interfaces[i].GQLType, interfaces[j].GQLType) == -1
	})

	return interfaces
}
