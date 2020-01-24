package federation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/formatter"
)

type federation struct {
	SDL      string
	Entities []*Entity
}

// New returns a federation plugin that injects
// federated directives and types into the schema
func New() plugin.Plugin {
	return &federation{}
}

// Name returns the plugin name
func (f *federation) Name() string {
	return "federation"
}

// MutateConfig mutates the configuration
func (f *federation) MutateConfig(cfg *config.Config) error {
	entityFields := map[string]config.TypeMapField{}
	for _, e := range f.Entities {
		entityFields[e.ResolverName] = config.TypeMapField{Resolver: true}
		for _, r := range e.Requires {
			if cfg.Models[e.Def.Name].Fields == nil {
				model := cfg.Models[e.Def.Name]
				model.Fields = map[string]config.TypeMapField{}
				cfg.Models[e.Def.Name] = model
			}
			cfg.Models[e.Def.Name].Fields[r.Name] = config.TypeMapField{Resolver: true}
		}
	}
	builtins := config.TypeMap{
		"_Service": {
			Model: config.StringList{
				"github.com/99designs/gqlgen/plugin/federation.Service",
			},
		},
		"_Any": {
			Model: config.StringList{"github.com/99designs/gqlgen/graphql.Map"},
		},
	}
	if len(entityFields) > 0 {
		builtins["Entity"] = config.TypeMapEntry{
			Fields: entityFields,
		}
	}
	for typeName, entry := range builtins {
		if cfg.Models.Exists(typeName) {
			return fmt.Errorf("%v already exists which must be reserved when Federation is enabled", typeName)
		}
		cfg.Models[typeName] = entry
	}
	cfg.Directives["external"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["requires"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["provides"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["key"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["extends"] = config.DirectiveConfig{SkipRuntime: true}

	return nil
}

// InjectSources creates a GraphQL Entity type with all
// the fields that had the @key directive
func (f *federation) InjectSources(cfg *config.Config) {
	cfg.AdditionalSources = append(cfg.AdditionalSources, f.getSource(false))

	f.setEntities(cfg)
	if len(f.Entities) == 0 {
		// It's unusual for a service not to have any entities, but
		// possible if it only exports top-level queries and mutations.
		return
	}

	s := "type Entity {\n"
	for _, e := range f.Entities {
		s += fmt.Sprintf("\t%s(%s: %s): %s!\n", e.ResolverName, e.Field.Name, e.Field.Type.String(), e.Def.Name)
	}
	s += "}"
	cfg.AdditionalSources = append(cfg.AdditionalSources, &ast.Source{Name: "entity.graphql", Input: s, BuiltIn: true})
}

// addEntityToSchema adds the _Entity Union and _entities query to schema.
// This is part of MutateSchema.
func (f *federation) addEntityToSchema(s *ast.Schema) {
	// --- Set _Entity Union ---
	union := &ast.Definition{
		Name:        "_Entity",
		Kind:        ast.Union,
		Description: "A union unifies all @entity types (TODO: interfaces)",
		Types:       []string{},
	}
	for _, ent := range f.Entities {
		union.Types = append(union.Types, ent.Def.Name)
		s.AddPossibleType("_Entity", ent.Def)
		s.AddImplements(ent.Def.Name, union)
	}
	s.Types[union.Name] = union

	// --- Set _entities query ---
	fieldDef := &ast.FieldDefinition{
		Name: "_entities",
		Type: ast.NonNullListType(ast.NamedType("_Entity", nil), nil),
		Arguments: ast.ArgumentDefinitionList{
			{
				Name: "representations",
				Type: ast.NonNullListType(ast.NonNullNamedType("_Any", nil), nil),
			},
		},
	}
	if s.Query == nil {
		s.Query = &ast.Definition{
			Kind: ast.Object,
			Name: "Query",
		}
		s.Types["Query"] = s.Query
	}
	s.Query.Fields = append(s.Query.Fields, fieldDef)
}

// addServiceToSchema adds the _Service type and _service query to schema.
// This is part of MutateSchema.
func (f *federation) addServiceToSchema(s *ast.Schema) {
	typeDef := &ast.Definition{
		Kind: ast.Object,
		Name: "_Service",
		Fields: ast.FieldList{
			&ast.FieldDefinition{
				Name: "sdl",
				Type: ast.NonNullNamedType("String", nil),
			},
		},
	}
	s.Types[typeDef.Name] = typeDef

	// --- set _service query ---
	_serviceDef := &ast.FieldDefinition{
		Name: "_service",
		Type: ast.NonNullNamedType("_Service", nil),
	}
	s.Query.Fields = append(s.Query.Fields, _serviceDef)
}

// MutateSchema creates types and query declarations
// that are required by the federation spec.
func (f *federation) MutateSchema(s *ast.Schema) error {
	// It's unusual for a service not to have any entities, but
	// possible if it only exports top-level queries and mutations.
	if len(f.Entities) > 0 {
		f.addEntityToSchema(s)
	}
	f.addServiceToSchema(s)
	return nil
}

func (f *federation) getSource(builtin bool) *ast.Source {
	return &ast.Source{
		Name: "federation.graphql",
		Input: `# Declarations as required by the federation spec 
# See: https://www.apollographql.com/docs/apollo-server/federation/federation-spec/

scalar _Any
scalar _FieldSet

directive @external on FIELD_DEFINITION
directive @requires(fields: _FieldSet!) on FIELD_DEFINITION
directive @provides(fields: _FieldSet!) on FIELD_DEFINITION
directive @key(fields: _FieldSet!) on OBJECT | INTERFACE
directive @extends on OBJECT
`,
		BuiltIn: builtin,
	}
}

// Entity represents a federated type
// that was declared in the GQL schema.
type Entity struct {
	Field        *ast.FieldDefinition
	FieldTypeGo  string // The Go representation of that field type
	ResolverName string // The resolver name, such as FindUserByID
	Def          *ast.Definition
	Requires     []*Requires
}

// Requires represents an @requires clause
type Requires struct {
	Name   string          // the name of the field
	Fields []*RequireField // the name of the sibling fields
}

// RequireField is similar to an entity but it is a field not
// an object
type RequireField struct {
	Name          string                // The same name as the type declaration
	NameGo        string                // The Go struct field name
	TypeReference *config.TypeReference // The Go representation of that field type
}

func (f *federation) GenerateCode(data *codegen.Data) error {
	sdl, err := f.getSDL(data.Config)
	if err != nil {
		return err
	}
	f.SDL = sdl
	if len(f.Entities) > 0 {
		data.Objects.ByName("Entity").Root = true
		for _, e := range f.Entities {
			obj := data.Objects.ByName(e.Def.Name)
			for _, f := range obj.Fields {
				if f.Name == e.Field.Name {
					e.FieldTypeGo = f.TypeReference.GO.String()
				}
				for _, r := range e.Requires {
					for _, rf := range r.Fields {
						if rf.Name == f.Name {
							rf.TypeReference = f.TypeReference
							rf.NameGo = f.GoFieldName
						}
					}
				}
			}
		}
	}

	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Filename:        "service.go",
		Data:            f,
		GeneratedHeader: true,
	})
}

func (f *federation) setEntities(cfg *config.Config) {
	schema, err := cfg.LoadSchema()
	if err != nil {
		panic(err)
	}
	for _, schemaType := range schema.Types {
		if schemaType.Kind == ast.Object {
			dir := schemaType.Directives.ForName("key") // TODO: interfaces
			if dir != nil {
				fieldName := dir.Arguments[0].Value.Raw // TODO: multiple arguments, and multiple keys
				if strings.Contains(fieldName, " ") {
					panic("only single fields are currently supported in @key declaration")
				}
				field := schemaType.Fields.ForName(fieldName)
				requires := []*Requires{}
				for _, f := range schemaType.Fields {
					dir := f.Directives.ForName("requires")
					if dir == nil {
						continue
					}
					fields := strings.Split(dir.Arguments[0].Value.Raw, " ")
					requireFields := []*RequireField{}
					for _, f := range fields {
						requireFields = append(requireFields, &RequireField{
							Name: f,
						})
					}
					requires = append(requires, &Requires{
						Name:   f.Name,
						Fields: requireFields,
					})
				}
				f.Entities = append(f.Entities, &Entity{
					Field:        field,
					Def:          schemaType,
					ResolverName: fmt.Sprintf("find%sBy%s", schemaType.Name, templates.ToGo(fieldName)),
					Requires:     requires,
				})
			}
		}
	}
}

func (f *federation) getSDL(c *config.Config) (string, error) {
	sources := []*ast.Source{f.getSource(true)}
	for _, filename := range c.SchemaFilename {
		filename = filepath.ToSlash(filename)
		var err error
		var schemaRaw []byte
		schemaRaw, err = ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
			os.Exit(1)
		}
		sources = append(sources, &ast.Source{Name: filename, Input: string(schemaRaw)})
	}
	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	formatter.NewFormatter(&buf).FormatSchema(schema)
	return buf.String(), nil
}

// Service is the service object that the
// generated.go file will return for the _service
// query
type Service struct {
	SDL string `json:"sdl"`
}
