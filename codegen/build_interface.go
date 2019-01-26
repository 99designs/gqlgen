package codegen

import (
	"go/types"

	"github.com/vektah/gqlparser/ast"
)

func (b *builder) buildInterface(typ *ast.Definition) *Interface {
	i := &Interface{
		Definition: b.NamedTypes[typ.Name],
		InTypemap:  b.Config.Models.UserDefined(typ.Name),
	}

	for _, implementor := range b.Schema.GetPossibleTypes(typ) {
		t := b.NamedTypes[implementor.Name]

		i.Implementors = append(i.Implementors, InterfaceImplementor{
			Definition:    t,
			ValueReceiver: b.isValueReceiver(b.NamedTypes[typ.Name], t),
		})
	}

	return i
}

func (b *builder) isValueReceiver(intf *TypeDefinition, implementor *TypeDefinition) bool {
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
