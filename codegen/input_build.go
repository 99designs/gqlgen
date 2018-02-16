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

func buildInputs(namedTypes NamedTypes, s *schema.Schema, prog *loader.Program) Objects {
	var inputs Objects

	for _, typ := range s.Types {
		switch typ := typ.(type) {
		case *schema.InputObject:
			input := buildInput(namedTypes, typ)

			def, err := findGoType(prog, input.Package, input.GoType)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
			}
			if def != nil {
				input.Marshaler = buildInputMarshaler(typ, def)
				bindObject(def.Type(), input)
			}

			inputs = append(inputs, input)
		}
	}

	sort.Slice(inputs, func(i, j int) bool {
		return strings.Compare(inputs[i].GQLType, inputs[j].GQLType) == -1
	})

	return inputs
}

func buildInput(types NamedTypes, typ *schema.InputObject) *Object {
	obj := &Object{NamedType: types[typ.TypeName()]}

	for _, field := range typ.Values {
		obj.Fields = append(obj.Fields, Field{
			GQLName: field.Name.Name,
			Type:    types.getType(field.Type),
			Object:  obj,
		})
	}
	return obj
}

// if user has implemented an UnmarshalGQL method on the input type manually, use it
// otherwise we will generate one.
func buildInputMarshaler(typ *schema.InputObject, def types.Object) *Ref {
	switch def := def.(type) {
	case *types.TypeName:
		namedType := def.Type().(*types.Named)
		for i := 0; i < namedType.NumMethods(); i++ {
			method := namedType.Method(i)
			if method.Name() == "UnmarshalGQL" {
				return nil
			}
		}
	}

	return &Ref{GoType: typ.Name}
}
