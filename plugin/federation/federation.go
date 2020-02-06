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
	builtins := config.TypeMap{
		"_Service": {
			Model: config.StringList{
				"github.com/99designs/gqlgen/plugin/federation/fedruntime.Service",
			},
		},
		"_Any": {
			Model: config.StringList{"github.com/99designs/gqlgen/graphql.Map"},
		},
		"_Entity": {
			Model: config.StringList{"github.com/99designs/gqlgen/plugin/federation/fedruntime.Entity"},
		},
		"TypeSafeEntityRoot": {
			Model: config.StringList{"github.com/99designs/gqlgen/plugin/federation/fedruntime.TypeSafeEntityRoot"},
		},
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

// injects the directives required for the user sources to compile, so we can reflect them in a second
func (f *federation) InjectSourcesEarly() *ast.Source {
	return &ast.Source{
		Name: "federation/directives.graphql",
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
		BuiltIn: true,
	}
}

// Injects
func (f *federation) InjectSourcesLate(schema *ast.Schema) *ast.Source {
	f.setEntities(schema)

	entities := ""
	resolvers := ""
	for i, e := range f.Entities {
		if i != 0 {
			entities += " | "
		}
		entities += e.Name

		resolverArgs := ""
		for _, field := range e.KeyFields {
			resolverArgs += fmt.Sprintf("%s: %s,", field.Field.Name, field.Field.Type.String())
		}
		resolvers += fmt.Sprintf("\t%s(%s): %s!\n", e.ResolverName, resolverArgs, e.Def.Name)

	}

	return &ast.Source{
		Name: "federation/entity.graphql",
		Input: `# a union of all types that use the @key directive
union _Entity = ` + entities + `

# fake type to build resolver interfaces for users to implement
type TypeSafeEntityRoot {
	` + resolvers + `
}

type _Service {
  sdl: String
}

extend type Query {
  _entities(representations: [_Any!]!): [_Entity]!
  _service: _Service!
}
`,
		BuiltIn: true,
	}
}

// Entity represents a federated type
// that was declared in the GQL schema.
type Entity struct {
	Name         string      // The same name as the type declaration
	KeyFields    []*KeyField // The fields declared in @key.
	ResolverName string      // The resolver name, such as FindUserByID
	Def          *ast.Definition
	Requires     []*Requires
}

type KeyField struct {
	Field         *ast.FieldDefinition
	TypeReference *config.TypeReference // The Go representation of that field type
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
		root := data.Objects.ByName("TypeSafeEntityRoot")
		root.Root = true

		// make the follow-schema layout generate resolver stubs next to the definition, not entites.schema
		for _, f := range root.Fields {
			f.Position = data.Schema.Types[f.Type.Name()].Position
		}

		for _, e := range f.Entities {
			obj := data.Objects.ByName(e.Def.Name)
			for _, field := range obj.Fields {
				// Storing key fields in a slice rather than a map
				// to preserve insertion order at the tradeoff of higher
				// lookup complexity.
				keyField := f.getKeyField(e.KeyFields, field.Name)
				if keyField != nil {
					keyField.TypeReference = field.TypeReference
				}
				for _, r := range e.Requires {
					for _, rf := range r.Fields {
						if rf.Name == field.Name {
							rf.TypeReference = field.TypeReference
							rf.NameGo = field.GoFieldName
						}
					}
				}
			}
		}
	}

	return templates.Render(templates.Options{
		PackageName:     data.Config.Federation.Package,
		Filename:        data.Config.Federation.Filename,
		Data:            f,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
	})
}

func (f *federation) getKeyField(keyFields []*KeyField, fieldName string) *KeyField {
	for _, field := range keyFields {
		if field.Field.Name == fieldName {
			return field
		}
	}
	return nil
}

func (f *federation) setEntities(schema *ast.Schema) {
	for _, schemaType := range schema.Types {
		if schemaType.Kind == ast.Object {
			dir := schemaType.Directives.ForName("key") // TODO: interfaces
			if dir != nil {
				if len(dir.Arguments) > 1 {
					panic("Multiple arguments are not currently supported in @key declaration.")
				}
				fieldName := dir.Arguments[0].Value.Raw // TODO: multiple arguments
				if strings.Contains(fieldName, "{") {
					panic("Nested fields are not currently supported in @key declaration.")
				}

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

				fieldNames := strings.Split(fieldName, " ")
				keyFields := make([]*KeyField, len(fieldNames))
				resolverName := fmt.Sprintf("find%sBy", schemaType.Name)
				for i, f := range fieldNames {
					field := schemaType.Fields.ForName(f)

					keyFields[i] = &KeyField{Field: field}
					if i > 0 {
						resolverName += "And"
					}
					resolverName += templates.ToGo(f)

				}

				f.Entities = append(f.Entities, &Entity{
					Name:         schemaType.Name,
					KeyFields:    keyFields,
					Def:          schemaType,
					ResolverName: resolverName,
					Requires:     requires,
				})
			}
		}
	}
}

func (f *federation) getSDL(c *config.Config) (string, error) {
	sources := []*ast.Source{f.InjectSourcesEarly()}
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
