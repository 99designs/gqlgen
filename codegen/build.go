package codegen

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

type builder struct {
	Config     *config.Config
	Schema     *ast.Schema
	SchemaStr  map[string]string
	Binder     *config.Binder
	Directives map[string]*Directive
	NamedTypes NamedTypes
}

func buildSchema(cfg *config.Config) (*Schema, error) {
	b := builder{
		Config: cfg,
	}

	var err error
	b.Schema, b.SchemaStr, err = cfg.LoadSchema()
	if err != nil {
		return nil, err
	}

	err = cfg.Check()
	if err != nil {
		return nil, err
	}

	b.Binder, err = b.Config.NewBinder()
	if err != nil {
		return nil, err
	}

	b.NamedTypes = NamedTypes{}

	for _, schemaType := range b.Schema.Types {
		b.NamedTypes[schemaType.Name], err = b.buildTypeDef(schemaType)
		if err != nil {
			return nil, errors.Wrap(err, "unable to build type definition")
		}
	}

	b.Directives, err = b.buildDirectives()
	if err != nil {
		return nil, err
	}

	s := Schema{
		Config:     cfg,
		Directives: b.Directives,
		Schema:     b.Schema,
		SchemaStr:  b.SchemaStr,
	}

	for _, schemaType := range b.Schema.Types {
		switch schemaType.Kind {
		case ast.Object:
			obj, err := b.buildObject(schemaType)
			if err != nil {
				return nil, errors.Wrap(err, "unable to build object definition")
			}

			s.Objects = append(s.Objects, obj)
		case ast.InputObject:
			input, err := b.buildObject(schemaType)
			if err != nil {
				return nil, errors.Wrap(err, "unable to build input definition")
			}

			s.Inputs = append(s.Inputs, input)

		case ast.Union, ast.Interface:
			s.Interfaces = append(s.Interfaces, b.buildInterface(schemaType))

		case ast.Enum:
			if enum := b.buildEnum(schemaType); enum != nil {
				s.Enums = append(s.Enums, *enum)
			}
		}
	}

	if err := b.injectIntrospectionRoots(&s); err != nil {
		return nil, err
	}

	sort.Slice(s.Objects, func(i, j int) bool {
		return s.Objects[i].Definition.GQLDefinition.Name < s.Objects[j].Definition.GQLDefinition.Name
	})

	sort.Slice(s.Inputs, func(i, j int) bool {
		return s.Inputs[i].Definition.GQLDefinition.Name < s.Inputs[j].Definition.GQLDefinition.Name
	})

	sort.Slice(s.Interfaces, func(i, j int) bool {
		return s.Interfaces[i].Definition.GQLDefinition.Name < s.Interfaces[j].Definition.GQLDefinition.Name
	})

	sort.Slice(s.Enums, func(i, j int) bool {
		return s.Enums[i].Definition.GQLDefinition.Name < s.Enums[j].Definition.GQLDefinition.Name
	})

	return &s, nil
}

func (b *builder) injectIntrospectionRoots(s *Schema) error {
	obj := s.Objects.ByName(b.Schema.Query.Name)
	if obj == nil {
		return fmt.Errorf("root query type must be defined")
	}

	typeType, err := b.Binder.FindObject("github.com/99designs/gqlgen/graphql/introspection", "Type")
	if err != nil {
		return errors.Wrap(err, "unable to find root Type introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference:  &TypeReference{b.NamedTypes["__Type"], types.NewPointer(typeType.Type()), ast.NamedType("__Schema", nil)},
		GQLName:        "__type",
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectType",
		Args: []*FieldArgument{
			{
				GQLName: "name",
				TypeReference: &TypeReference{
					b.NamedTypes["String"],
					types.Typ[types.String],
					ast.NamedType("String", nil),
				},
				Object: &Object{},
			},
		},
		Object: obj,
	})

	schemaType, err := b.Binder.FindObject("github.com/99designs/gqlgen/graphql/introspection", "Schema")
	if err != nil {
		return errors.Wrap(err, "unable to find root Schema introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference:  &TypeReference{b.NamedTypes["__Schema"], types.NewPointer(schemaType.Type()), ast.NamedType("__Schema", nil)},
		GQLName:        "__schema",
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectSchema",
		Object:         obj,
	})

	return nil
}
