package modelgen

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

func renameLegacyID(f *Field) {
	if f.Name == "_id" {
		f.GoName = "LegacyID"
	}
}

// Not parallel: LoadSchema writes package-level caches in internal/code.
func TestGeneratedMembers(t *testing.T) {
	cases := []struct {
		name     string
		schema   string
		models   config.TypeMap // gqlgen.yml models
		template string         // model_template
		plugin   func(*Plugin)
		err      string
		getter   string // a getter the generated file must contain
	}{
		{
			name:   "fields",
			schema: `type Foo { _id: ID! id: ID! }`,
			err: `type "Foo": field "_id" and field "id" both have Go name ID; ` +
				`rename one with @goField(name: ...) or gqlgen.yml fieldName`,
		},
		{
			name:   "@goField name of another field",
			schema: `type Foo { a: ID! @goField(name: "B") b: ID! }`,
			err:    `type "Foo": field "a" and field "b" both have Go name B`,
		},
		{
			name:   "input fields",
			schema: `input FooInput { _id: ID id: ID }`,
			err:    `type "FooInput": field "_id" and field "id" both have Go name ID`,
		},
		{
			name:   "extra field",
			schema: `type Foo { id: ID! }`,
			models: config.TypeMap{"Foo": {ExtraFields: map[string]config.ModelExtraField{
				"ID": {Type: "string"},
			}}},
			err: `type "Foo": field "id" and field "ID" both have Go name ID`,
		},
		{
			name: "interface getters",
			schema: `
				interface Node { _id: ID! id: ID! }
				type Foo implements Node { _id: ID! @goField(name: "LegacyID") id: ID! }`,
			err: `type "Node": the getter for Node._id and the getter for Node.id both have Go name GetID`,
		},
		{
			name: "interface marker",
			schema: `
				interface Bar { id: ID! }
				type Foo implements Bar { id: ID! isBar: Boolean! }`,
			err: `type "Foo": field "isBar" and the method marking interface "Bar" both have Go name IsBar`,
		},
		{
			name: "getter",
			schema: `
				interface Bar { id: ID! }
				type Foo implements Bar { id: ID! getID: ID! }`,
			err: `type "Foo": field "getID" and the getter for Bar.id both have Go name GetID`,
		},
		{
			name: "getters for different fields with one Go name",
			schema: `
				interface A { _id: ID! }
				interface B { id: ID! }
				type Foo implements A & B { _id: ID! @goField(name: "LegacyID") id: ID! }`,
			err: `type "Foo": the getter for A._id and the getter for B.id both have Go name GetID; ` +
				`rename one with @goField(name: ...) or gqlgen.yml fieldName`,
		},
		{
			name: "getters for different fields and types with one Go name",
			schema: `
				interface A { _id: Int! }
				interface B { id: ID! }
				type Foo implements A & B { _id: Int! @goField(name: "LegacyID") id: ID! }`,
			err: `type "Foo": the getter for A._id and the getter for B.id both have Go name GetID`,
		},
		{
			name: "getters for one field with two types",
			schema: `
				interface A { id: ID! }
				interface B { id: ID }
				type Foo implements A & B { id: ID! }`,
			err: `type "Foo": the getter for A.id returns string but the getter for B.id returns *string; ` +
				`give the interface fields one type`,
		},
		{
			name: "getter without a field",
			schema: `
				interface A { id: ID! }
				type Foo implements A { id: ID! }`,
			models: config.TypeMap{"Foo": {Fields: map[string]config.TypeMapField{
				"id": {Resolver: true},
			}}},
			err: `type "Foo": the getter for A.id has no field to return; ` +
				`generate field "id" on "Foo", or set omit_getters`,
		},
		{
			name: "getter for a renamed field",
			schema: `
				interface A { _id: ID! }
				type Foo implements A { _id: ID! @goField(name: "LegacyID") }`,
			getter: `func (this Foo) GetID() string { return this.LegacyID }`,
		},
		{
			name: "getter for a renamed field next to its old Go name",
			schema: `
				interface A { _id: ID! }
				type Foo implements A { _id: ID! @goField(name: "LegacyID") id: ID! }`,
			getter: `func (this Foo) GetID() string { return this.LegacyID }`,
		},
		{
			name: "getter shared by interfaces",
			schema: `
				interface A { id: ID! }
				interface B { id: ID! }
				type Foo implements A & B { id: ID! }`,
			getter: `func (this Foo) GetID() string { return this.ID }`,
		},
		{
			name: "embedded struct",
			schema: `
				interface Node @goEmbedInterface { id: ID! }
				type Foo implements Node { id: ID! baseNode: String! }`,
			err: `type "Foo": embedded BaseNode and field "baseNode" both have Go name BaseNode`,
		},
		{
			name: "getter for a list",
			schema: `
				interface A { ids: [ID!] }
				type Foo implements A { ids: [ID!] }`,
			getter: `func (this Foo) GetIds() []string { if this.Ids == nil { return nil } ` +
				`interfaceSlice := make([]string, 0, len(this.Ids)) ` +
				`for _, concrete := range this.Ids { interfaceSlice = append(interfaceSlice, concrete) } ` +
				`return interfaceSlice }`,
		},
		{
			name: "getter description",
			schema: `
				interface A { "The key." id: ID! }
				type Foo implements A { id: ID! }`,
			getter: `// The key. func (this Foo) GetID() string { return this.ID }`,
		},
		{
			name: "getters after a bound interface",
			schema: `
				interface Bound { id: ID! }
				interface A { id: ID! }
				type Foo implements Bound & A { id: ID! }`,
			models: config.TypeMap{"Bound": {Model: config.StringList{"fmt.Stringer"}}},
			getter: `func (this Foo) GetID() string { return this.ID }`,
		},
		{
			name: "getters after a shared one",
			schema: `
				interface A { id: ID! }
				interface B { id: ID! name: String! }
				type Foo implements A & B { id: ID! name: String! }`,
			getter: `func (this Foo) GetName() string { return this.Name }`,
		},
		{
			name: "getters with a custom template",
			schema: `
				interface A { id: ID! }
				type Foo implements A { id: ID! }`,
			template: "testdata/customModelTemplate.gotpl",
			getter:   `func (this Foo) GetID() string { return this.ID }`,
		},
		{
			name:   "omitted resolver field",
			schema: `type Foo { _id: ID! id: ID! }`,
			models: config.TypeMap{"Foo": {Fields: map[string]config.TypeMapField{
				"_id": {Resolver: true},
			}}},
		},
		{
			name:   "names differing in case",
			schema: `type Foo { legacyId: ID! @goField(name: "LegacyId") legacyID: ID! }`,
		},
		{
			name:   "renamed by the field hook",
			schema: `type Foo { _id: ID! id: ID! }`,
			plugin: func(p *Plugin) {
				p.FieldHook = func(td *ast.Definition, fd *ast.FieldDefinition, f *Field) (*Field, error) {
					renameLegacyID(f)
					return DefaultFieldMutateHook(td, fd, f)
				}
			},
		},
		{
			name:   "renamed by the build hook",
			schema: `type Foo { _id: ID! id: ID! }`,
			plugin: func(p *Plugin) {
				p.MutateHook = func(b *ModelBuild) *ModelBuild {
					for _, model := range b.Models {
						for _, f := range model.Fields {
							renameLegacyID(f)
						}
					}
					return b
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.OmitResolverFields = true
			cfg.Model = config.PackageConfig{
				Filename:      filepath.Join(t.TempDir(), "models.go"),
				Package:       "model",
				ModelTemplate: tc.template,
			}
			maps.Copy(cfg.Models, tc.models)
			cfg.Sources = []*ast.Source{{
				Name: "schema.graphql", Input: `
				directive @goField(name: String) on FIELD_DEFINITION | INPUT_FIELD_DEFINITION
				directive @goEmbedInterface on INTERFACE
				type Query { ok: Boolean }` + tc.schema,
			}}
			require.NoError(t, cfg.Init())
			p := New().(*Plugin)
			if tc.plugin != nil {
				tc.plugin(p)
			}

			err := p.MutateConfig(cfg)

			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			if tc.getter != "" {
				src, err := os.ReadFile(cfg.Model.Filename)
				require.NoError(t, err)
				// gofmt aligns one-line functions, so compare words.
				require.Contains(t, strings.Join(strings.Fields(string(src)), " "), tc.getter)
			}
		})
	}
}
