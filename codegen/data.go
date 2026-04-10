package codegen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

// Data is a unified model of the code to be generated. Plugins may modify this structure to do
// things like implement
// resolvers or directives automatically (eg grpc, validation)
type Data struct {
	Config *config.Config
	Schema *ast.Schema
	// If a schema is broken up into multiple Data instance, each representing part of the schema,
	// AllDirectives should contain the directives for the entire schema. Directives() can
	// then be used to get the directives that were defined in this Data instance's sources.
	// If a single Data instance is used for the entire schema, AllDirectives and Directives()
	// will be identical.
	// AllDirectives should rarely be used directly.
	AllDirectives   DirectiveList
	Objects         Objects
	Inputs          Objects
	Interfaces      map[string]*Interface
	ReferencedTypes map[string]*config.TypeReference
	ComplexityRoots map[string]*Object

	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	AugmentedSources []AugmentedSource
	Plugins          []any

	// SkipLocationDirectives suppresses generation of location directive middleware
	// (_fieldMiddleware, _queryMiddleware, etc.) in per-schema builds.
	// In follow-schema layout these are generated in the root file instead.
	SkipLocationDirectives bool
}

func (d *Data) HasEmbeddableSources() bool {
	hasEmbeddableSources := false
	for _, s := range d.AugmentedSources {
		if s.Embeddable {
			hasEmbeddableSources = true
		}
	}
	return hasEmbeddableSources
}

func (d *Data) HasBatchResolverFields() bool {
	for _, obj := range d.Objects {
		if obj.Root {
			continue
		}
		for _, field := range obj.Fields {
			if field.IsBatch() {
				return true
			}
		}
	}
	return false
}

// AugmentedSource contains extra information about graphql schema files which is not known directly
// from the Config.Sources data
type AugmentedSource struct {
	// path relative to Config.Exec.Filename
	RelativePath string
	Embeddable   bool
	BuiltIn      bool
	Source       string
}

type builder struct {
	Config     *config.Config
	Schema     *ast.Schema
	Binder     *config.Binder
	Directives map[string]*Directive
}

// Get only the directives which should have a user provided definition on server instantiation
func (d *Data) UserDirectives() DirectiveList {
	res := DirectiveList{}
	directives := d.Directives()
	for k, directive := range directives {
		if directive.Implementation == nil {
			res[k] = directive
		}
	}
	return res
}

// Get only the directives which should have a statically provided definition
func (d *Data) BuiltInDirectives() DirectiveList {
	res := DirectiveList{}
	directives := d.Directives()
	for k, directive := range directives {
		if directive.Implementation != nil {
			res[k] = directive
		}
	}
	return res
}

// Get only the directives which are defined in the config's sources.
func (d *Data) Directives() DirectiveList {
	res := DirectiveList{}
	for k, directive := range d.AllDirectives {
		for _, s := range d.Config.Sources {
			if directive.Position.Src.Name == s.Name {
				res[k] = directive
				break
			}
		}
	}
	return res
}

// DirectiveArgs returns directive argument parser functions that are not generated
// by any per-schema build. A directive's args are orphaned when its source file
// contains no type definitions (Objects, Inputs, Interfaces, ReferencedTypes),
// meaning no per-schema build exists for that file.
func (d *Data) DirectiveArgs() map[string][]*FieldArgument {
	sourcesWithTypes := map[string]bool{}
	for _, o := range d.Objects {
		if o.Position != nil && o.Position.Src != nil {
			sourcesWithTypes[o.Position.Src.Name] = true
		}
	}
	for _, in := range d.Inputs {
		if in.Position != nil && in.Position.Src != nil {
			sourcesWithTypes[in.Position.Src.Name] = true
		}
	}
	for _, inf := range d.Interfaces {
		if inf.Position != nil && inf.Position.Src != nil {
			sourcesWithTypes[inf.Position.Src.Name] = true
		}
	}
	for _, rt := range d.ReferencedTypes {
		if rt.Definition != nil && rt.Definition.Position != nil && rt.Definition.Position.Src != nil {
			sourcesWithTypes[rt.Definition.Position.Src.Name] = true
		}
	}

	ret := map[string][]*FieldArgument{}
	for _, directive := range d.AllDirectives {
		if len(directive.Args) == 0 {
			continue
		}
		if directive.Position != nil && directive.Position.Src != nil &&
			!sourcesWithTypes[directive.Position.Src.Name] {
			ret[directive.ArgsFunc()] = directive.Args
		}
	}
	return ret
}

func BuildData(cfg *config.Config, plugins ...any) (*Data, error) {
	cfg.ReloadAllPackages()

	b := builder{
		Config: cfg,
		Schema: cfg.Schema,
	}

	b.Binder = b.Config.NewBinder()

	var err error
	b.Directives, err = b.buildDirectives()
	if err != nil {
		return nil, err
	}

	dataDirectives := make(map[string]*Directive)
	for name, d := range b.Directives {
		if !d.SkipRuntime {
			dataDirectives[name] = d
		}
	}

	s := Data{
		Config:        cfg,
		AllDirectives: dataDirectives,
		Schema:        b.Schema,
		Interfaces:    map[string]*Interface{},
		Plugins:       plugins,
	}

	for _, schemaType := range b.Schema.Types {
		switch schemaType.Kind {
		case ast.Object:
			obj, err := b.buildObject(schemaType)
			if err != nil {
				return nil, fmt.Errorf("unable to build object definition: %w", err)
			}

			s.Objects = append(s.Objects, obj)
		case ast.InputObject:
			input, err := b.buildObject(schemaType)
			if err != nil {
				return nil, fmt.Errorf("unable to build input definition: %w", err)
			}

			s.Inputs = append(s.Inputs, input)

		case ast.Union, ast.Interface:
			s.Interfaces[schemaType.Name], err = b.buildInterface(schemaType)
			if err != nil {
				return nil, fmt.Errorf("unable to bind to interface: %w", err)
			}
		}
	}

	if s.Schema.Query != nil {
		s.QueryRoot = s.Objects.ByName(s.Schema.Query.Name)
	} else {
		return nil, errors.New("query entry point missing")
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
		return s.Objects[i].Name < s.Objects[j].Name
	})

	sort.Slice(s.Inputs, func(i, j int) bool {
		return s.Inputs[i].Name < s.Inputs[j].Name
	})

	if b.Binder.SawInvalid {
		// if we have a syntax error, show it
		err := cfg.Packages.Errors()
		if len(err) > 0 {
			return nil, err
		}

		// otherwise show a generic error message
		return nil, errors.New(
			"invalid types were encountered while traversing the go source code, this probably means the invalid code generated isnt correct. add try adding -v to debug",
		)
	}
	var sources []*ast.Source
	sources, err = SerializeTransformedSchema(cfg.Schema, cfg.Sources)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transformed schema: %w", err)
	}

	aSources := []AugmentedSource{}
	for _, s := range sources {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		outputDir := cfg.Exec.Dir()
		sourcePath := filepath.Join(wd, s.Name)
		relative, err := filepath.Rel(outputDir, sourcePath)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to compute path of %s relative to %s: %w",
				sourcePath,
				outputDir,
				err,
			)
		}
		relative = filepath.ToSlash(relative)
		embeddable := true
		if strings.HasPrefix(relative, "..") || s.BuiltIn {
			embeddable = false
		}
		aSources = append(aSources, AugmentedSource{
			RelativePath: relative,
			Embeddable:   embeddable,
			BuiltIn:      s.BuiltIn,
			Source:       s.Input,
		})
	}
	s.AugmentedSources = aSources

	return &s, nil
}

func (b *builder) injectIntrospectionRoots(s *Data) error {
	obj := s.Objects.ByName(b.Schema.Query.Name)
	if obj == nil {
		return errors.New("root query type must be defined")
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
