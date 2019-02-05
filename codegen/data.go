package codegen

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

// Data is a unified model of the code to be generated. Plugins may modify this structure to do things like implement
// resolvers or directives automatically (eg grpc, validation)
type Data struct {
	Config     *config.Config
	Schema     *ast.Schema
	SchemaStr  map[string]string
	Directives map[string]*Directive
	Objects    Objects
	Inputs     Objects
	Interfaces map[string]*Interface

	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
}

type builder struct {
	Config     *config.Config
	Schema     *ast.Schema
	SchemaStr  map[string]string
	Binder     *config.Binder
	Directives map[string]*Directive
	NamedTypes NamedTypes
}

func BuildData(cfg *config.Config) (*Data, error) {
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

	cfg.InjectBuiltins(b.Schema)

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

	s := Data{
		Config:     cfg,
		Directives: b.Directives,
		Schema:     b.Schema,
		SchemaStr:  b.SchemaStr,
		Interfaces: map[string]*Interface{},
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
			s.Interfaces[schemaType.Name] = b.buildInterface(schemaType)
		}
	}

	if s.Schema.Query != nil {
		s.QueryRoot = s.Objects.ByName(s.Schema.Query.Name)
	} else {
		return nil, fmt.Errorf("query entry point missing")
	}

	if s.Schema.Mutation != nil {
		s.MutationRoot = s.Objects.ByName(s.Schema.Mutation.Name)
	}

	if s.Schema.Subscription != nil {
		s.SubscriptionRoot = s.Objects.ByName(s.Schema.Subscription.Name)
	}

	if err := b.injectIntrospectionRoots(&s); err != nil {
		return nil, err
	}

	sort.Slice(s.Objects, func(i, j int) bool {
		return s.Objects[i].Definition.Name < s.Objects[j].Definition.Name
	})

	sort.Slice(s.Inputs, func(i, j int) bool {
		return s.Inputs[i].Definition.Name < s.Inputs[j].Definition.Name
	})

	return &s, nil
}

func (b *builder) injectIntrospectionRoots(s *Data) error {
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
