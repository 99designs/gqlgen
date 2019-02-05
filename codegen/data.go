package codegen

import (
	"fmt"
	"sort"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

// Data is a unified model of the code to be generated. Plugins may modify this structure to do things like implement
// resolvers or directives automatically (eg grpc, validation)
type Data struct {
	Config          *config.Config
	Schema          *ast.Schema
	SchemaStr       map[string]string
	Directives      map[string]*Directive
	Objects         Objects
	Inputs          Objects
	Interfaces      map[string]*Interface
	ReferencedTypes map[string]*config.TypeReference

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

	b.Binder, err = b.Config.NewBinder(b.Schema)
	if err != nil {
		return nil, err
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

	s.ReferencedTypes, err = b.buildTypes()
	if err != nil {
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

	typeType, err := b.Binder.TypeReference(ast.NamedType("__Type", nil))
	if err != nil {
		return errors.Wrap(err, "unable to find root Type introspection type")
	}
	stringRef, err := b.Binder.TypeReference(ast.NonNullNamedType("String", nil))
	if err != nil {
		return errors.Wrap(err, "unable to find root string type reference")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference: typeType,
		FieldDefinition: &ast.FieldDefinition{
			Name: "__type",
		},
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectType",
		Args: []*FieldArgument{
			{
				ArgumentDefinition: &ast.ArgumentDefinition{
					Name: "name",
				},
				TypeReference: stringRef,
				Object:        &Object{},
			},
		},
		Object: obj,
	})

	schemaType, err := b.Binder.TypeReference(ast.NamedType("__Schema", nil))
	if err != nil {
		return errors.Wrap(err, "unable to find root Schema introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference: schemaType,
		FieldDefinition: &ast.FieldDefinition{
			Name: "__schema",
		},
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectSchema",
		Object:         obj,
	})

	return nil
}
