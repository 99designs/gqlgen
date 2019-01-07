package codegen

import (
	"go/types"
	"sort"

	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (g *Generator) buildInterfaces(types NamedTypes, prog *loader.Program) []*Interface {
	var interfaces []*Interface
	for _, typ := range g.schema.Types {
		if typ.Kind == ast.Union || typ.Kind == ast.Interface {
			interfaces = append(interfaces, g.buildInterface(types, typ, prog))
		}
	}

	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].GQLType < interfaces[j].GQLType
	})

	return interfaces
}

func (g *Generator) buildInterface(types NamedTypes, typ *ast.Definition, prog *loader.Program) *Interface {
	i := &Interface{TypeDefinition: types[typ.Name]}

	for _, implementor := range g.schema.GetPossibleTypes(typ) {
		t := types[implementor.Name]

		i.Implementors = append(i.Implementors, InterfaceImplementor{
			TypeDefinition: t,
			ValueReceiver:  g.isValueReceiver(types[typ.Name], t, prog),
		})
	}

	return i
}

func (g *Generator) isValueReceiver(intf *TypeDefinition, implementor *TypeDefinition, prog *loader.Program) bool {
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
