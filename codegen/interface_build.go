package codegen

import (
	"fmt"
	"go/types"
	"os"
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func buildInterfaces(types NamedTypes, s *schema.Schema, prog *loader.Program) []*Interface {
	var interfaces []*Interface
	for _, typ := range s.Types {
		switch typ := typ.(type) {
		case *schema.Union, *schema.Interface:
			interfaces = append(interfaces, buildInterface(types, typ, prog))
		default:
			continue
		}
	}

	sort.Slice(interfaces, func(i, j int) bool {
		return strings.Compare(interfaces[i].GQLType, interfaces[j].GQLType) == -1
	})

	return interfaces
}

func buildInterface(types NamedTypes, typ schema.NamedType, prog *loader.Program) *Interface {
	switch typ := typ.(type) {

	case *schema.Union:
		i := &Interface{NamedType: types[typ.TypeName()]}

		for _, implementor := range typ.PossibleTypes {
			t := types[implementor.TypeName()]

			i.Implementors = append(i.Implementors, InterfaceImplementor{
				NamedType:     t,
				ValueReceiver: isValueReceiver(types[typ.Name], t, prog),
			})
		}

		return i

	case *schema.Interface:
		i := &Interface{NamedType: types[typ.TypeName()]}

		for _, implementor := range typ.PossibleTypes {
			t := types[implementor.TypeName()]

			i.Implementors = append(i.Implementors, InterfaceImplementor{
				NamedType:     t,
				ValueReceiver: isValueReceiver(types[typ.Name], t, prog),
			})
		}

		return i
	default:
		panic(fmt.Errorf("unknown interface %#v", typ))
	}
}

func isValueReceiver(intf *NamedType, implementor *NamedType, prog *loader.Program) bool {
	interfaceType := findGoInterface(prog, intf.Package, intf.GoType)
	implementorType := findGoNamedType(prog, implementor.Package, implementor.GoType)

	if interfaceType == nil || implementorType == nil {
		return true
	}

	for i := 0; i < interfaceType.NumMethods(); i++ {
		intfMethod := interfaceType.Method(i)

		implMethod := findMethod(implementorType, intfMethod.Name())
		if implMethod == nil {
			fmt.Fprintf(os.Stderr, "missing method %s on %s\n", intfMethod.Name(), implementor.GoType)
			return false
		}

		sig := implMethod.Type().(*types.Signature)
		if _, isPtr := sig.Recv().Type().(*types.Pointer); isPtr {
			return false
		}
	}

	return true
}
