package codegen

import (
	"go/types"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

func (cfg *Config) buildInputs(namedTypes NamedTypes, prog *loader.Program, imports Imports) (Objects, error) {
	var inputs Objects

	for _, typ := range cfg.schema.Types {
		switch typ := typ.(type) {
		case *schema.InputObject:
			input := buildInput(namedTypes, typ)

			def, err := findGoType(prog, input.Package, input.GoType)
			if err != nil {
				return nil, errors.Wrap(err, "cannot find type")
			}
			if def != nil {
				input.Marshaler = buildInputMarshaler(typ, def)
				bindObject(def.Type(), input, imports)
			}

			inputs = append(inputs, input)
		}
	}

	sort.Slice(inputs, func(i, j int) bool {
		return strings.Compare(inputs[i].GQLType, inputs[j].GQLType) == -1
	})

	return inputs, nil
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
