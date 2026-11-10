//go:generate go run ../../testdata/gqlgen.go -config testdata/perentity/gqlgen.yml
package federation

import (
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/internal/code"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentity"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentity/generated"
)

// injectFederation runs the plugin's schema and config steps on schema, as
// api.Generate does, after configure (if not nil) edits the default config as
// gqlgen.yml would, and returns the error from the late injection or the
// config step, which are the steps under test. Not parallel-safe: LoadSchema
// writes package-level caches in internal/code.
func injectFederation(
	t *testing.T,
	schema string,
	configure func(*config.Config),
) (*Federation, *config.Config, error) {
	t.Helper()

	cfg := config.DefaultConfig()
	// MutateConfig writes federation_keys.go beside the models, with the
	// package cache cfg.Init would create.
	cfg.Model = config.PackageConfig{
		Filename: filepath.Join(t.TempDir(), "models_gen.go"),
		Package:  "model",
	}
	cfg.Packages = code.NewPackages()
	if configure != nil {
		configure(cfg)
	}
	f, err := New(2, cfg)
	require.NoError(t, err)
	early, err := f.InjectSourcesEarly()
	require.NoError(t, err)
	cfg.Sources = early
	cfg.Sources = append(cfg.Sources, &ast.Source{
		Name: "schema.graphql",
		Input: `directive @entityResolver(multi: Boolean, requires: String) on OBJECT | INTERFACE
			directive @goField(name: String) on FIELD_DEFINITION
			type Query { ok: Boolean }
			` + schema,
	})
	require.NoError(t, cfg.LoadSchema())

	late, err := f.InjectSourcesLate(cfg.Schema)
	if err != nil {
		return f, nil, err
	}
	cfg.Sources = append(cfg.Sources, late...)
	require.NoError(t, cfg.LoadSchema())
	if err := f.MutateConfig(cfg); err != nil {
		return f, nil, err
	}
	return f, cfg, nil
}

// perEntity configures per-entity resolvers, as entity_resolver_per_entity
// does in gqlgen.yml.
func perEntity(cfg *config.Config) {
	cfg.Federation.Options = map[string]bool{optionEntityResolverPerEntity: true}
}

func TestPerEntityOptIn(t *testing.T) {
	cases := []struct {
		name    string
		options map[string]bool // federation.options
		want    bool
	}{
		{name: "per-key by default", want: false},
		{
			name:    "per-entity with the option",
			options: map[string]bool{optionEntityResolverPerEntity: true},
			want:    true,
		},
		{
			name:    "per-key with the option off",
			options: map[string]bool{optionEntityResolverPerEntity: false},
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, _, err := injectFederation(t, `type Port @key(fields: "id") { id: ID! }`,
				func(cfg *config.Config) { cfg.Federation.Options = tc.options })

			require.NoError(t, err)
			require.Len(t, f.Entities, 1)
			require.Equal(t, tc.want, f.Entities[0].PerEntity)
		})
	}
}

func TestPerEntityErrors(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		models config.TypeMap    // gqlgen.yml models
		names  map[string]string // federation.representation_names
		err    string
	}{
		{
			name: "representation name taken",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				input PortRepresentation { other: String }`,
			err: `entity "Port": type "PortRepresentation" has the same Go name as ` +
				`representation PortRepresentation; rename "PortRepresentation", ` +
				`or name the representation of "Port" in federation.representation_names`,
		},
		{
			name: "representation Go name taken",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				input portRepresentation { other: String }`,
			err: `entity "Port": type "portRepresentation" has the same Go name as ` +
				`representation PortRepresentation; rename "portRepresentation", ` +
				`or name the representation of "Port" in federation.representation_names`,
		},
		{
			name: "nested representation Go name taken",
			schema: `
				type Ship @key(fields: "owner { id }") { owner: Owner! }
				type Owner { id: ID! }
				input ownerRepresentation { other: String }`,
			err: `entity "Ship": type "ownerRepresentation" has the same Go name as ` +
				`representation OwnerRepresentation; rename "ownerRepresentation", ` +
				`or name the representation of "Owner" in federation.representation_names`,
		},
		{
			name: "interface with @requires",
			schema: `
				interface Node @key(fields: "id") {
					id: ID!
					size: Int! @external
					label: String! @requires(fields: "size")
				}
				type Dock implements Node @key(fields: "id") {
					id: ID!
					size: Int! @external
					label: String! @requires(fields: "size")
				}`,
			err: `entity "Node": @requires on an interface entity is not supported ` +
				`with the entity_resolver_per_entity option, and federation does not allow it`,
		},
		{
			name: "named representation Go name taken",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				input PortKey { other: String }`,
			names: map[string]string{"Port": "portKey"},
			err: `entity "Port": type "PortKey" has the same Go name as representation portKey, ` +
				`named in federation.representation_names; choose another name`,
		},
		{
			name: "representations named alike",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				type Ship @key(fields: "id") { id: ID! }`,
			names: map[string]string{"Ship": "PortRepresentation"},
			err: `entity "Ship": the representations of "Port" and "Ship" both have Go name ` +
				`PortRepresentation; give one another name in federation.representation_names`,
		},
		{
			name:   "name for a type without a representation",
			schema: `type Port @key(fields: "id") { id: ID! }`,
			names:  map[string]string{"Owner": "OwnerKey"},
			err:    `federation.representation_names: no representation is generated for "Owner"`,
		},
		{
			name:   "MatchedKey taken by an extra field",
			schema: `type Port @key(fields: "id") { id: ID! }`,
			models: config.TypeMap{
				"PortRepresentation": {ExtraFields: map[string]config.ModelExtraField{
					"MatchedKey": {Type: "int"},
				}},
			},
			err: `models.PortRepresentation.extraFields.MatchedKey is reserved: ` +
				`gqlgen sets it to the @key the representation matched`,
		},
		{
			name: "entities sharing a Go name",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				type _Port @key(fields: "id") { id: ID! }`,
			err: `entities "Port" and "_Port" both generate entity resolver find_Port`,
		},
		{
			name: "key through a list",
			schema: `
				type Ship @key(fields: "owners { id }") { owners: [Owner!]! }
				type Owner { id: ID! }`,
			err: `entity "Ship": key field Ship.owners has type [Owner!]!; federation ` +
				`does not allow list or union fields in @key`,
		},
		{
			name: "key field of union type",
			schema: `
				type Ship @key(fields: "owner") { owner: Owner! }
				type Person { id: ID! }
				union Owner = Person`,
			err: `entity "Ship": key field Ship.owner has type Owner!; federation ` +
				`does not allow list or union fields in @key`,
		},
		{
			name: "preloaded @requires through an inline fragment",
			schema: `
				type Ship @key(fields: "id") @entityResolver(requires: "preloaded") {
					id: ID!
					owner: Owner @external
					crew: Int @requires(fields: "owner { ... on Person { id } }")
				}
				type Person { id: ID! }
				union Owner = Person`,
			err: `entity "Ship": @requires path through inline fragment "... on Person" on ` +
				`Ship.owner is not supported with the entity_resolver_per_entity option`,
		},
		{
			name: "preloaded @requires of no field",
			schema: `
				type Ship @key(fields: "id") @entityResolver(requires: "preloaded") {
					id: ID!
					crew: Int @requires(fields: "nope")
				}`,
			err: `entity "Ship": @requires field "nope" is not a field of Ship`,
		},
		{
			name: "key field of object type",
			schema: `
				type Ship @key(fields: "owner") { owner: Owner! }
				type Owner { id: ID! }`,
			err: `entity "Ship": key field Ship.owner has object type Owner; select its fields`,
		},
		{
			name: "key field of interface type",
			schema: `
				type Ship @key(fields: "owner") { owner: Owner! }
				interface Owner { id: ID! }`,
			err: `entity "Ship": key field Ship.owner has interface type Owner; select its fields`,
		},
		{
			name: "MatchedKey type name taken by a type",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				type PortMatchedKey { id: ID! }`,
			err: `type "PortMatchedKey" has the same Go name as PortMatchedKey, ` +
				`the MatchedKey type of entity "Port"; rename "PortMatchedKey"`,
		},
		{
			name: "MatchedKey constant name taken by a type",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				scalar PortMatchedKeyID`,
			err: `type "PortMatchedKeyID" has the same Go name as PortMatchedKeyID, ` +
				`the MatchedKey constant of @key(fields: "id") on "Port"; rename "PortMatchedKeyID"`,
		},
		{
			name: "MatchedKey constant name taken by an enum value",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				enum PortMatched { KEY_ID }`,
			err: `enum value PortMatched.KEY_ID has the same Go name as PortMatchedKeyID, ` +
				`the MatchedKey constant of @key(fields: "id") on "Port"; ` +
				`rename PortMatched.KEY_ID`,
		},
		{
			name:   "MatchedKey type name taken by a representation",
			schema: `type Port @key(fields: "id") { id: ID! }`,
			names:  map[string]string{"Port": "PortMatchedKey"},
			err: `representation PortMatchedKey has the same Go name as PortMatchedKey, ` +
				`the MatchedKey type of entity "Port"; give the representation of "Port" ` +
				`another name in federation.representation_names`,
		},
		{
			name: "preloaded @requires field of object type",
			schema: `
				type Ship @key(fields: "id") @entityResolver(requires: "preloaded") {
					id: ID!
					owner: Owner @external
					crew: Int @requires(fields: "owner")
				}
				type Owner { id: ID! }`,
			err: `entity "Ship": @requires field Ship.owner has object type Owner; select its fields`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := injectFederation(t, tc.schema, func(cfg *config.Config) {
				perEntity(cfg)
				maps.Copy(cfg.Models, tc.models)
				cfg.Federation.RepresentationNames = tc.names
			})

			require.EqualError(t, err, tc.err)
		})
	}
}

// fieldsSDL renders a definition's fields, with their arguments, as SDL.
func fieldsSDL(def *ast.Definition) []string {
	fields := make([]string, 0, len(def.Fields))
	for _, field := range def.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}
		args := make([]string, 0, len(field.Arguments))
		for _, arg := range field.Arguments {
			args = append(args, arg.Name+": "+arg.Type.String())
		}
		sdl := field.Name
		if len(args) > 0 {
			sdl += "(" + strings.Join(args, ", ") + ")"
		}
		fields = append(fields, sdl+": "+field.Type.String())
	}
	return fields
}

func TestPerEntitySchema(t *testing.T) {
	cases := []struct {
		name     string
		schema   string
		models   config.TypeMap    // gqlgen.yml models
		names    map[string]string // federation.representation_names
		resolver []string
		inputs   map[string][]string
		goNames  map[string]map[string]string // representation field Go names
	}{
		{
			name: "single resolver for colliding keys",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String! @goField(name: "LegacyID")
				id: ID!
			}`,
			resolver: []string{"findPort(rep: PortRepresentation!): Port!"},
			inputs:   map[string][]string{"PortRepresentation": {"_id: String", "id: ID"}},
			goNames: map[string]map[string]string{
				"PortRepresentation": {"_id": "LegacyID", "id": "ID"},
			},
		},
		{
			name: "single resolver for keys whose paths join to one name",
			schema: `
				type Port @key(fields: "id") @key(fields: "i { d }") {
					id: ID!
					i: Owner!
				}
				type Owner { d: ID! }`,
			resolver: []string{"findPort(rep: PortRepresentation!): Port!"},
			inputs: map[string][]string{
				"PortRepresentation":  {"id: ID", "i: OwnerRepresentation"},
				"OwnerRepresentation": {"d: ID"},
			},
		},
		{
			name: "gqlgen.yml fieldName on the owner",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String!
				id: ID!
			}`,
			models: config.TypeMap{"Port": {Fields: map[string]config.TypeMapField{
				"_id": {FieldName: "LegacyID"},
			}}},
			goNames: map[string]map[string]string{
				"PortRepresentation": {"_id": "LegacyID", "id": "ID"},
			},
		},
		{
			name: "gqlgen.yml fieldName on the representation wins",
			schema: `
				type Ship @key(fields: "owner { id }") { owner: Owner! }
				type Owner { id: ID! @goField(name: "OwnerKey") }`,
			models: config.TypeMap{"OwnerRepresentation": {Fields: map[string]config.TypeMapField{
				"id": {FieldName: "ChosenByUser"},
			}}},
			goNames: map[string]map[string]string{
				"ShipRepresentation":  {"owner": "Owner"},
				"OwnerRepresentation": {"id": "ChosenByUser"},
			},
		},
		{
			name: "batch resolver with nested key",
			schema: `
				type Ship @key(fields: "id owner { id }") @entityResolver(multi: true) {
					id: ID!
					owner: Owner!
				}
				type Owner { id: ID! name: String }`,
			resolver: []string{"findShip(reps: [ShipRepresentation!]!): [Ship]"},
			inputs: map[string][]string{
				"ShipRepresentation":  {"id: ID", "owner: OwnerRepresentation"},
				"OwnerRepresentation": {"id: ID"},
			},
		},
		{
			name: "key through an interface field",
			schema: `
				type Ship @key(fields: "owner { id }") { owner: Owner! }
				interface Owner { id: ID! }
				type Person implements Owner { id: ID! name: String }`,
			resolver: []string{"findShip(rep: ShipRepresentation!): Ship!"},
			inputs: map[string][]string{
				"ShipRepresentation":  {"owner: OwnerRepresentation"},
				"OwnerRepresentation": {"id: ID"},
			},
		},
		{
			name: "keys sharing fields list them once",
			schema: `
				type Ship @key(fields: "id owner { id }") @key(fields: "id owner { name }") {
					id: ID!
					owner: Owner!
				}
				type Owner { id: ID! name: String! }`,
			inputs: map[string][]string{
				"ShipRepresentation":  {"id: ID", "owner: OwnerRepresentation"},
				"OwnerRepresentation": {"id: ID", "name: String"},
			},
		},
		{
			name: "nested representation merges fields across entities",
			schema: `
				type Boat @key(fields: "owner { name }") { owner: Owner! }
				type Ship @key(fields: "owner { id }") { owner: Owner! }
				type Owner { id: ID! name: String! }`,
			resolver: []string{
				"findBoat(rep: BoatRepresentation!): Boat!",
				"findShip(rep: ShipRepresentation!): Ship!",
			},
			inputs: map[string][]string{
				"BoatRepresentation":  {"owner: OwnerRepresentation"},
				"ShipRepresentation":  {"owner: OwnerRepresentation"},
				"OwnerRepresentation": {"name: String", "id: ID"},
			},
		},
		{
			name: "entities whose per-key resolver names collide",
			schema: `
				type Port @key(fields: "iDByX") { iDByX: ID! }
				type PortByID @key(fields: "x") { x: ID! }`,
			resolver: []string{
				"findPort(rep: PortRepresentation!): Port!",
				"findPortByID(rep: PortByIDRepresentation!): PortByID!",
			},
		},
		{
			name: "named representations",
			schema: `
				type Ship @key(fields: "id owner { id }") { id: ID! owner: Owner! }
				type Owner { id: ID! }`,
			names:    map[string]string{"Ship": "ShipKey", "Owner": "OwnerKey"},
			resolver: []string{"findShip(rep: ShipKey!): Ship!"},
			inputs: map[string][]string{
				"ShipKey":  {"id: ID", "owner: OwnerKey"},
				"OwnerKey": {"id: ID"},
			},
		},
		{
			name: "named representation beside a type with the default name",
			schema: `
				type Port @key(fields: "id") { id: ID! }
				input PortRepresentation { other: String }`,
			names:    map[string]string{"Port": "PortKey"},
			resolver: []string{"findPort(rep: PortKey!): Port!"},
			inputs: map[string][]string{
				"PortKey":            {"id: ID"},
				"PortRepresentation": {"other: String"},
			},
		},
		{
			name: "interface",
			schema: `
				interface Node @key(fields: "id") { id: ID! }
				type Dock implements Node @key(fields: "id") { id: ID! }`,
			resolver: []string{
				"findDock(rep: DockRepresentation!): Dock!",
				"findNode(rep: NodeRepresentation!): Node!",
			},
			inputs: map[string][]string{
				"DockRepresentation": {"id: ID"},
				"NodeRepresentation": {"id: ID"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cfg, err := injectFederation(t, tc.schema, func(cfg *config.Config) {
				perEntity(cfg)
				maps.Copy(cfg.Models, tc.models)
				cfg.Federation.RepresentationNames = tc.names
			})

			require.NoError(t, err)
			if tc.resolver != nil {
				require.Equal(t, tc.resolver, fieldsSDL(cfg.Schema.Types["Entity"]))
			}
			for name, fields := range tc.inputs {
				def := cfg.Schema.Types[name]
				require.NotNil(t, def, "input %s missing from schema", name)
				require.Equal(t, ast.InputObject, def.Kind)
				require.Equal(t, fields, fieldsSDL(def), "fields of %s", name)
			}
			for name, want := range tc.goNames {
				got := make(map[string]string)
				for field, entry := range cfg.Models[name].Fields {
					got[field] = entry.FieldName
				}
				require.Equal(t, want, got, "Go names of %s", name)
			}
		})
	}
}

// TestPerEntityPreloadedSchema checks which @requires fields a representation
// carries: under preloaded, every non-computed one, once, beside the keys.
func TestPerEntityPreloadedSchema(t *testing.T) {
	cases := []struct {
		name    string
		schema  string
		options map[string]bool     // federation.options beside per-entity
		types   map[string][]string // fields of each named definition
	}{
		{
			name: "default strategy carries keys only",
			schema: `type Planet @key(fields: "name") {
				name: String!
				diameter: Int @external
				size: Int @requires(fields: "diameter")
			}`,
			types: map[string][]string{"PlanetRepresentation": {"name: String"}},
		},
		{
			name: "preloaded carries @requires",
			schema: `type Planet @key(fields: "name") @entityResolver(requires: "preloaded") {
				name: String!
				diameter: Int! @external
				size: Int @requires(fields: "diameter")
			}`,
			types: map[string][]string{
				"PlanetRepresentation": {"name: String", "diameter: Int"},
			},
		},
		{
			name: "preloaded nested @requires",
			schema: `
				type Comet @key(fields: "name") @entityResolver(requires: "preloaded") {
					name: String!
					orbit: Orbit @external
					tag: Int @requires(fields: "orbit { period }")
				}
				type Orbit { period: Int! radius: Int }`,
			types: map[string][]string{
				"CometRepresentation": {"name: String", "orbit: OrbitRepresentation"},
				"OrbitRepresentation": {"period: Int"},
			},
		},
		{
			name: "preloaded @requires through a list",
			schema: `
				type Ship @key(fields: "id") @entityResolver(requires: "preloaded") {
					id: ID!
					owners: [Owner!]! @external
					crew: Int @requires(fields: "owners { id }")
				}
				type Owner { id: ID! name: String }`,
			types: map[string][]string{
				"ShipRepresentation":  {"id: ID", "owners: [OwnerRepresentation!]"},
				"OwnerRepresentation": {"id: ID"},
			},
		},
		{
			name: "preloaded @requires of a key field lists it once",
			schema: `type Planet @key(fields: "name") @entityResolver(requires: "preloaded") {
				name: String!
				size: Int @requires(fields: "name")
			}`,
			types: map[string][]string{"PlanetRepresentation": {"name: String"}},
		},
		{
			name: "computed @requires stays off the representation",
			schema: `type Planet @key(fields: "name") @entityResolver(requires: "preloaded") {
				name: String!
				diameter: Int @external
				mass: Int @external
				size: Int @requires(fields: "diameter") @computedRequires
				density: Int @requires(fields: "mass")
			}`,
			types: map[string][]string{
				"PlanetRepresentation": {"name: String", "mass: Int"},
			},
		},
		{
			name: "preloaded_requires option on a multi entity without @requires",
			schema: `type Planet @key(fields: "name") @entityResolver(multi: true) {
				name: String!
			}`,
			options: map[string]bool{optionPreloadedRequires: true},
			types:   map[string][]string{"PlanetRepresentation": {"name: String"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cfg, err := injectFederation(t, tc.schema, func(cfg *config.Config) {
				perEntity(cfg)
				maps.Copy(cfg.Federation.Options, tc.options)
				// The computed case needs it; it changes nothing else here.
				cfg.CallArgumentDirectivesWithNull = true
			})

			require.NoError(t, err)
			for name, fields := range tc.types {
				require.Equal(t, fields, fieldsSDL(cfg.Schema.Types[name]), "fields of %s", name)
			}
		})
	}
}

// TestPerEntityRepresentationLeavesSchema checks that a representation's list
// field is a copy: renaming its element type must not rename the entity's.
func TestPerEntityRepresentationLeavesSchema(t *testing.T) {
	cfg := config.DefaultConfig()
	perEntity(cfg)
	f, err := New(2, cfg)
	require.NoError(t, err)
	early, err := f.InjectSourcesEarly()
	require.NoError(t, err)
	cfg.Sources = early
	cfg.Sources = append(cfg.Sources, &ast.Source{Name: "schema.graphql", Input: `
		directive @entityResolver(requires: String) on OBJECT
		type Query { ok: Boolean }
		type Ship @key(fields: "id") @entityResolver(requires: "preloaded") {
			id: ID!
			owners: [Owner!]! @external
			crew: Int @requires(fields: "owners { id }")
		}
		type Owner { id: ID! }`})
	require.NoError(t, cfg.LoadSchema())

	_, err = f.InjectSourcesLate(cfg.Schema)

	require.NoError(t, err)
	owners := cfg.Schema.Types["Ship"].Fields.ForName("owners")
	require.Equal(t, "[Owner!]!", owners.Type.String())
}

func TestPerEntityMatchedKeyField(t *testing.T) {
	_, cfg, err := injectFederation(t, `
		type Ship @key(fields: "id owner { id }") { id: ID! owner: Owner! }
		type Owner { id: ID! }`, func(cfg *config.Config) {
		perEntity(cfg)
		cfg.Models["ShipRepresentation"] = config.TypeMapEntry{
			ExtraFields: map[string]config.ModelExtraField{"Note": {Type: "string"}},
		}
	})

	require.NoError(t, err)
	extra := cfg.Models["ShipRepresentation"].ExtraFields
	require.ElementsMatch(t, []string{"MatchedKey", "Note"}, slices.Collect(maps.Keys(extra)),
		"a root representation gets MatchedKey beside the user's extra fields")
	require.Equal(t, cfg.Model.ImportPath()+".ShipMatchedKey", extra["MatchedKey"].Type,
		"MatchedKey has the entity's key type, declared in the model package")
	require.Empty(t, cfg.Models["OwnerRepresentation"].ExtraFields,
		"a nested representation never matches a key itself")
}

func TestPerEntityMatchedKeyConsts(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		models config.TypeMap // gqlgen.yml models
		want   []string       // MatchedKeyConst per @key, in order
	}{
		{
			name:   "one key",
			schema: `type Port @key(fields: "id") { id: ID! }`,
			want:   []string{"PortMatchedKeyID"},
		},
		{
			name: "@goField names the constant",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String! @goField(name: "LegacyID")
				id: ID!
			}`,
			want: []string{"PortMatchedKeyLegacyID", "PortMatchedKeyID"},
		},
		{
			name:   "gqlgen.yml fieldName names the constant",
			schema: `type Port @key(fields: "code") { code: String! }`,
			models: config.TypeMap{"Port": {Fields: map[string]config.TypeMapField{
				"code": {FieldName: "PortCode"},
			}}},
			want: []string{"PortMatchedKeyPortCode"},
		},
		{
			name: "compound and nested keys",
			schema: `
				type Ship @key(fields: "id owner { id }") @key(fields: "name") {
					id: ID!
					owner: Owner!
					name: String!
				}
				type Owner { id: ID! }`,
			want: []string{"ShipMatchedKeyIDAndOwnerID", "ShipMatchedKeyName"},
		},
		{
			name: "keys whose names collide get a suffix",
			schema: `
				type Port @key(fields: "id") @key(fields: "i { d }") {
					id: ID!
					i: Owner!
				}
				type Owner { d: ID! }`,
			want: []string{"PortMatchedKeyID", "PortMatchedKeyID2"},
		},
		{
			name: "interface entity",
			schema: `interface Node @key(fields: "id") { id: ID! }
				type Thing implements Node @key(fields: "id") { id: ID! }`,
			want: []string{"NodeMatchedKeyID"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, _, err := injectFederation(t, tc.schema, func(cfg *config.Config) {
				perEntity(cfg)
				maps.Copy(cfg.Models, tc.models)
			})

			require.NoError(t, err)
			entity := f.Entities[0]
			var got []string
			for _, resolver := range entity.Resolvers {
				got = append(got, resolver.MatchedKeyConst)
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestPerEntityMatchedKeysFile(t *testing.T) {
	generated := templates.GeneratedCodeHeader + "\n\npackage model\n"
	cases := []struct {
		name      string
		perEntity bool
		existing  string // federation_keys.go before generating; "" for none
		want      string // federation_keys.go after; "" for none
	}{
		{
			name:      "declares each entity's type and constants",
			perEntity: true,
			want: templates.GeneratedCodeHeader + `

package model

// PortMatchedKey is the @key a Port representation matched, which
// its MatchedKey field reports: one of the constants below, each holding
// the key's fields as gqlgen prints them.
type PortMatchedKey string

const (
	PortMatchedKeyLegacyID     PortMatchedKey = "_id"
	PortMatchedKeyIDAndOwnerID PortMatchedKey = "id owner { id }"
)
`,
		},
		{
			name:     "removes the file gqlgen generated when no entity is per-entity",
			existing: generated,
		},
		{
			name:     "keeps a hand-written file",
			existing: "package model\n",
			want:     "package model\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var filename string
			_, _, err := injectFederation(t, `
				type Port @key(fields: "_id") @key(fields: "id owner { id }") {
					_id: String! @goField(name: "LegacyID")
					id: ID!
					owner: Owner!
				}
				type Owner { id: ID! }`, func(cfg *config.Config) {
				if tc.perEntity {
					perEntity(cfg)
				}
				filename = filepath.Join(filepath.Dir(cfg.Model.Filename), "federation_keys.go")
				if tc.existing != "" {
					require.NoError(t, os.WriteFile(filename, []byte(tc.existing), 0o600))
				}
			})

			require.NoError(t, err)
			got, err := os.ReadFile(filename)
			if tc.want == "" {
				require.ErrorIs(t, err, fs.ErrNotExist)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, string(got))
		})
	}
}

func TestRepresentationMatchedKeyType(t *testing.T) {
	pkg := types.NewPackage("example.com/model", "model")
	keyType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "PortMatchedKey", nil), types.Typ[types.String], nil)
	otherKeyType := types.NewNamed(types.NewTypeName(token.NoPos,
		types.NewPackage("example.com/other", "other"), "PortMatchedKey", nil),
		types.Typ[types.String], nil)
	representation := func(fields ...*types.Var) types.Type {
		return types.NewNamed(types.NewTypeName(token.NoPos, pkg, "PortRepresentation", nil),
			types.NewStruct(fields, nil), nil)
	}
	matchedKey := func(typ types.Type) *types.Var {
		return types.NewField(token.NoPos, pkg, "MatchedKey", typ, false)
	}
	cases := []struct {
		name string
		rep  types.Type
		err  string
	}{
		{name: "generated field", rep: representation(matchedKey(keyType))},
		{
			name: "no field",
			rep:  representation(),
			err:  "representation example.com/model.PortRepresentation has no MatchedKey field",
		},
		{
			name: "bound struct with a string field",
			rep:  representation(matchedKey(types.Typ[types.String])),
			err: "example.com/model.PortRepresentation.MatchedKey has type string; " +
				"declare it as the generated example.com/model.PortMatchedKey",
		},
		{
			name: "same name in another package",
			rep:  representation(matchedKey(otherKeyType)),
			err: "example.com/model.PortRepresentation.MatchedKey has type " +
				"example.com/other.PortMatchedKey; " +
				"declare it as the generated example.com/model.PortMatchedKey",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := representationMatchedKeyType(tc.rep, "example.com/model", "PortMatchedKey")

			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Same(t, keyType, got)
		})
	}
}

func TestPerEntityResolver(t *testing.T) {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: &perentity.Resolver{},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	type entity struct {
		LegacyID   string `json:"_id"`
		ID         string `json:"id"`
		ResolvedBy string `json:"resolvedBy"`
		Size       int    `json:"size"`
	}
	cases := []struct {
		name  string
		query string
		reps  []map[string]any
		want  []entity
	}{
		{
			name:  "both keys sent matches the first declared",
			query: "Port { _id id resolvedBy size }",
			reps:  []map[string]any{{"__typename": "Port", "_id": "legacy", "id": "p1", "size": 3}},
			want:  []entity{{LegacyID: "legacy", ResolvedBy: "_id", Size: 3}},
		},
		{
			name:  "single by _id lands on LegacyID",
			query: "Port { _id id resolvedBy size }",
			reps:  []map[string]any{{"__typename": "Port", "_id": "legacy", "size": 3}},
			want:  []entity{{LegacyID: "legacy", ResolvedBy: "_id", Size: 3}},
		},
		{
			name:  "single by id",
			query: "Port { _id id resolvedBy size }",
			reps:  []map[string]any{{"__typename": "Port", "id": "p1", "size": 4}},
			want:  []entity{{ID: "p1", ResolvedBy: "id", Size: 4}},
		},
		{
			name:  "batch with mixed keys",
			query: "Ship { _id id resolvedBy }",
			reps: []map[string]any{
				{"__typename": "Ship", "id": "s1", "owner": map[string]any{"id": "o1"}},
				{"__typename": "Ship", "_id": "legacy"},
			},
			want: []entity{
				{ID: "s1", ResolvedBy: "id owner { id }"},
				{LegacyID: "legacy", ResolvedBy: "_id"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp struct {
				Entities []entity `json:"_entities"`
			}

			err := c.Post(entityQuery([]string{tc.query}), &resp,
				client.Var("representations", tc.reps))

			require.NoError(t, err)
			require.Equal(t, tc.want, resp.Entities)
		})
	}

	t.Run("nested key reaches the batch resolver", func(t *testing.T) {
		var resp struct {
			Entities []struct {
				Owner struct {
					ID string `json:"id"`
				} `json:"owner"`
			} `json:"_entities"`
		}

		err := c.Post(entityQuery([]string{"Ship { owner { id } }"}), &resp,
			client.Var("representations", []map[string]any{
				{"__typename": "Ship", "id": "s1", "owner": map[string]any{"id": "o1"}},
			}))

		require.NoError(t, err)
		require.Len(t, resp.Entities, 1)
		require.Equal(t, "o1", resp.Entities[0].Owner.ID)
	})

	t.Run("key through an interface field", func(t *testing.T) {
		var resp struct {
			Entities []struct {
				Holder struct {
					Typename string `json:"__typename"`
					ID       string `json:"id"`
				} `json:"holder"`
				ResolvedBy string `json:"resolvedBy"`
			} `json:"_entities"`
		}

		err := c.Post(entityQuery([]string{"Berth { holder { __typename id } resolvedBy }"}), &resp,
			client.Var("representations", []map[string]any{
				{"__typename": "Berth", "holder": map[string]any{"id": "c1"}},
			}))

		require.NoError(t, err)
		require.Len(t, resp.Entities, 1)
		require.Equal(t, "Captain", resp.Entities[0].Holder.Typename)
		require.Equal(t, "c1", resp.Entities[0].Holder.ID)
		require.Equal(t, "holder { id }", resp.Entities[0].ResolvedBy)
	})

	t.Run("interface resolves to its implementation", func(t *testing.T) {
		type dock struct {
			Typename string `json:"__typename"`
			ID       string `json:"id"`
			Berths   int    `json:"berths"`
		}
		var resp struct {
			Entities []dock `json:"_entities"`
		}

		err := c.Post(entityQuery([]string{"Node { __typename id }", "Dock { berths }"}), &resp,
			client.Var("representations", []map[string]any{
				{"__typename": "Node", "id": "n1"},
				{"__typename": "Dock", "id": "d1"},
			}))

		require.NoError(t, err)
		require.Equal(t, []dock{
			{Typename: "Dock", ID: "n1", Berths: 2},
			{Typename: "Dock", ID: "d1", Berths: 1},
		}, resp.Entities, "findNode serves Node; findDockByID serves Dock")
	})

	t.Run("representation without a key", func(t *testing.T) {
		var resp map[string]any

		err := c.Post(entityQuery([]string{"Port { id }"}), &resp,
			client.Var("representations", []map[string]any{{"__typename": "Port", "size": 1}}))

		require.ErrorContains(t, err, `missing Key Field \"_id\" for Port`)
	})
}

func TestPerEntityIgnoredWithoutResolvers(t *testing.T) {
	_, cfg, err := injectFederation(t, `
		type Port @key(fields: "id") { id: ID! @external }`, perEntity)

	require.NoError(t, err)
	require.Nil(t, cfg.Schema.Types["Entity"], "an all-external entity gets no resolver")
	require.Nil(t, cfg.Schema.Types["PortRepresentation"])
}
