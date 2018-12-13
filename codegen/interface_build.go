package codegen

import (
	"go/types"
	"sort"

	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (cfg *Generator) buildInterfaces(types NamedTypes, prog *loader.Program) []*Interface {
	var interfaces []*Interface
	for _, typ := range cfg.schema.Types {
		if typ.Kind == ast.Union || typ.Kind == ast.Interface {
			interfaces = append(interfaces, cfg.buildInterface(types, typ, prog))
		}
	}

	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].GQLType < interfaces[j].GQLType
	})

	return interfaces
}

func (cfg *Generator) buildInterface(types NamedTypes, typ *ast.Definition, prog *loader.Program) *Interface {
	i := &Interface{TypeDefinition: types[typ.Name]}

	for _, implementor := range cfg.schema.GetPossibleTypes(typ) {
		t := types[implementor.Name]

		i.Implementors = append(i.Implementors, InterfaceImplementor{
			TypeDefinition: t,
			ValueReceiver:  cfg.isValueReceiver(types[typ.Name], t, prog),
		})
	}

	return i
}

func (cfg *Generator) isValueReceiver(intf *TypeDefinition, implementor *TypeDefinition, prog *loader.Program) bool {
	interfaceType, err := findGoInterface(prog, intf.Package, intf.GoType)
	if interfaceType == nil || err != nil {
		return true
	}

	implementorType, err := findGoNamedType(prog, implementor.Package, implementor.GoType)
	if implementorType == nil || err != nil {
		return true
	}

	return types.Implements(implementorType, interfaceType)
}
