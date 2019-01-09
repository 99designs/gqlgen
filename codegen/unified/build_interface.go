package unified

import (
	"go/types"
	"strings"

	"github.com/vektah/gqlparser/ast"
)

func (g *Schema) buildInterface(typ *ast.Definition) *Interface {
	i := &Interface{
		Definition: g.NamedTypes[typ.Name],
		InTypemap:  g.Config.Models.UserDefined(typ.Name),
	}

	for _, implementor := range g.Schema.GetPossibleTypes(typ) {
		t := g.NamedTypes[implementor.Name]

		i.Implementors = append(i.Implementors, InterfaceImplementor{
			Definition:    t,
			ValueReceiver: g.isValueReceiver(g.NamedTypes[typ.Name], t),
		})
	}

	return i
}

func (g *Schema) isValueReceiver(intf *TypeDefinition, implementor *TypeDefinition) bool {
	interfaceType, err := findGoInterface(intf.GoType)
	if interfaceType == nil || err != nil {
		return true
	}

	implementorType, err := findGoNamedType(implementor.GoType)
	if implementorType == nil || err != nil {
		return true
	}

	return types.Implements(implementorType, interfaceType)
}

// take a string in the form github.com/package/blah.TypeReference and split it into package and type
func pkgAndType(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", name
	}

	return normalizeVendor(strings.Join(parts[:len(parts)-1], ".")), parts[len(parts)-1]
}
