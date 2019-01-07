package codegen

import (
	"sort"

	"go/types"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (g *Generator) buildInputs(namedTypes NamedTypes, prog *loader.Program) (Objects, error) {
	var inputs Objects

	for _, typ := range g.schema.Types {
		switch typ.Kind {
		case ast.InputObject:
			input, err := g.buildInput(namedTypes, typ)
			if err != nil {
				return nil, err
			}

			if _, isMap := input.GoType.(*types.Map); !isMap {
				bindErrs := bindObject(input, g.StructTag)
				if len(bindErrs) > 0 {
					return nil, bindErrs
				}
			}

			inputs = append(inputs, input)
		}
	}

	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].GQLType < inputs[j].GQLType
	})

	return inputs, nil
}

func (g *Generator) buildInput(types NamedTypes, typ *ast.Definition) (*Object, error) {
	obj := &Object{TypeDefinition: types[typ.Name]}
	typeEntry, entryExists := g.Models[typ.Name]

	for _, field := range typ.Fields {
		dirs, err := g.getDirectives(field.Directives)
		if err != nil {
			return nil, err
		}
		newField := Field{
			GQLName:       field.Name,
			TypeReference: types.getType(field.Type),
			Object:        obj,
			Directives:    dirs,
		}

		if entryExists {
			if typeField, ok := typeEntry.Fields[field.Name]; ok {
				newField.GoFieldName = typeField.FieldName
			}
		}

		if field.DefaultValue != nil {
			var err error
			newField.Default, err = field.DefaultValue.Value(nil)
			if err != nil {
				return nil, errors.Errorf("default value for %s.%s is not valid: %s", typ.Name, field.Name, err.Error())
			}
		}

		if !newField.TypeReference.IsInput && !newField.TypeReference.IsScalar {
			return nil, errors.Errorf("%s cannot be used as a field of %s. only input and scalar types are allowed", newField.GQLType, obj.GQLType)
		}

		obj.Fields = append(obj.Fields, newField)

	}
	dirs, err := g.getDirectives(typ.Directives)
	if err != nil {
		return nil, err
	}
	obj.Directives = dirs

	return obj, nil
}
