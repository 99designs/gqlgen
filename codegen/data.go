package codegen

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/formatter"
)

// Data is a unified model of the code to be generated. Plugins may modify this structure to do things like implement
// resolvers or directives automatically (eg grpc, validation)
type Data struct {
	Config          *config.Config
	Schema          *ast.Schema
	SchemaStr       map[string]string
	Directives      DirectiveList
	Objects         Objects
	Inputs          Objects
	Interfaces      map[string]*Interface
	ReferencedTypes map[string]*config.TypeReference
	ComplexityRoots map[string]*Object

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

type SchemaMutator interface {
	MutateSchema(s *ast.Schema) error
}

func BuildData(cfg *config.Config, plugins []SchemaMutator) (*Data, error) {
	b := builder{
		Config: cfg,
	}

	var err error
	b.Schema, err = cfg.LoadSchema()
	if err != nil {
		return nil, err
	}

	err = cfg.Check()
	if err != nil {
		return nil, err
	}

	err = cfg.Autobind(b.Schema)
	if err != nil {
		return nil, err
	}

	cfg.InjectBuiltins(b.Schema)

	for _, p := range plugins {
		err = p.MutateSchema(b.Schema)
		if err != nil {
			return nil, fmt.Errorf("error running MutateSchema: %v", err)
		}
	}

	b.Binder, err = b.Config.NewBinder(b.Schema)
	if err != nil {
		return nil, err
	}

	b.Directives, err = b.buildDirectives()
	if err != nil {
		return nil, err
	}

	dataDirectives := make(map[string]*Directive)
	for name, d := range b.Directives {
		if !d.Builtin {
			dataDirectives[name] = d
		}
	}

	s := Data{
		Config:     cfg,
		Directives: dataDirectives,
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

	s.ReferencedTypes = b.buildTypes()

	sort.Slice(s.Objects, func(i, j int) bool {
		return s.Objects[i].Definition.Name < s.Objects[j].Definition.Name
	})

	sort.Slice(s.Inputs, func(i, j int) bool {
		return s.Inputs[i].Definition.Name < s.Inputs[j].Definition.Name
	})

	if b.Binder.SawInvalid {
		// if we have a syntax error, show it
		if len(b.Binder.PkgErrors) > 0 {
			return nil, b.Binder.PkgErrors
		}

		// otherwise show a generic error message
		return nil, fmt.Errorf("invalid types were encountered while traversing the go source code, this probably means the invalid code generated isnt correct. add try adding -v to debug")
	}

	var buf bytes.Buffer
	formatter.NewFormatter(&buf).FormatSchema(b.Schema)
	s.SchemaStr = map[string]string{"schema.graphql": buf.String()}

	return &s, nil
}

func (b *builder) injectIntrospectionRoots(s *Data) error {
	obj := s.Objects.ByName(b.Schema.Query.Name)
	if obj == nil {
		return fmt.Errorf("root query type must be defined")
	}

	__type, err := b.buildField(obj, &ast.FieldDefinition{
		Name: "__type",
		Type: ast.NamedType("__Type", nil),
		Arguments: []*ast.ArgumentDefinition{
			{
				Name: "name",
				Type: ast.NonNullNamedType("String", nil),
			},
		},
	})
	if err != nil {
		return err
	}

	__schema, err := b.buildField(obj, &ast.FieldDefinition{
		Name: "__schema",
		Type: ast.NamedType("__Schema", nil),
	})
	if err != nil {
		return err
	}

	obj.Fields = append(obj.Fields, __type, __schema)

	return nil
}
