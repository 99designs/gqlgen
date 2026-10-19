//go:generate go run ../../testdata/gqlgen.go -config testdata/perentity/gqlgen.yml
package federation

import (
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentity"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentity/generated"
)

// injectFederation runs the plugin's schema and config steps on schema, as
// api.Generate does, with models as the gqlgen.yml models, and returns the
// error from the late injection, which is the step under test. Not
// parallel-safe: LoadSchema writes package-level caches in internal/code.
func injectFederation(
	t *testing.T,
	schema string,
	models config.TypeMap,
) (*Federation, *config.Config, error) {
	t.Helper()

	cfg := config.DefaultConfig()
	maps.Copy(cfg.Models, models)
	f, err := New(2, cfg)
	require.NoError(t, err)
	early, err := f.InjectSourcesEarly()
	require.NoError(t, err)
	cfg.Sources = early
	cfg.Sources = append(cfg.Sources, &ast.Source{
		Name: "schema.graphql",
		Input: `directive @entityResolver(multi: Boolean, requires: String, perEntity: Boolean) on OBJECT | INTERFACE
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
	require.NoError(t, f.MutateConfig(cfg))
	return f, cfg, nil
}

func TestPerEntityOptIn(t *testing.T) {
	cases := []struct {
		name   string
		schema string
		want   bool
	}{
		{
			name:   "per-key by default",
			schema: `type Port @key(fields: "id") { id: ID! }`,
			want:   false,
		},
		{
			name:   "opted in",
			schema: `type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! }`,
			want:   true,
		},
		{
			name:   "explicitly off",
			schema: `type Port @key(fields: "id") @entityResolver(perEntity: false) { id: ID! }`,
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, _, err := injectFederation(t, tc.schema, nil)

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
		err    string
	}{
		{
			name: "preloaded requires",
			schema: `type Port @key(fields: "id")
				@entityResolver(perEntity: true, multi: true, requires: "preloaded") {
				id: ID!
			}`,
			err: `entity "Port": @entityResolver(perEntity: true) cannot be combined`,
		},
		{
			name: "representation name taken",
			schema: `
				type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! }
				input PortRepresentation { other: String }`,
			err: `entity "Port": type "PortRepresentation" has the same Go name as ` +
				`representation PortRepresentation; rename "PortRepresentation"`,
		},
		{
			name: "representation Go name taken",
			schema: `
				type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! }
				input portRepresentation { other: String }`,
			err: `entity "Port": type "portRepresentation" has the same Go name as ` +
				`representation PortRepresentation; rename "portRepresentation"`,
		},
		{
			name: "nested representation Go name taken",
			schema: `
				type Ship @key(fields: "owner { id }") @entityResolver(perEntity: true) { owner: Owner! }
				type Owner { id: ID! }
				input ownerRepresentation { other: String }`,
			err: `entity "Ship": type "ownerRepresentation" has the same Go name as ` +
				`representation OwnerRepresentation; rename "ownerRepresentation"`,
		},
		{
			name: "interface with @requires",
			schema: `
				interface Node @key(fields: "id") @entityResolver(perEntity: true) {
					id: ID!
					size: Int! @external
					label: String! @requires(fields: "size")
				}
				type Dock implements Node @key(fields: "id") {
					id: ID!
					size: Int! @external
					label: String! @requires(fields: "size")
				}`,
			err: `entity "Node": @entityResolver(perEntity: true) on an interface does not support @requires`,
		},
		{
			name: "key through a list",
			schema: `
				type Ship @key(fields: "owners { id }") @entityResolver(perEntity: true) { owners: [Owner!]! }
				type Owner { id: ID! }`,
			err: `entity "Ship": key path through list field Ship.owners is not supported`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := injectFederation(t, tc.schema, nil)

			require.ErrorContains(t, err, tc.err)
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
		models   config.TypeMap // gqlgen.yml models
		resolver []string
		inputs   map[string][]string
		goNames  map[string]map[string]string // representation field Go names
	}{
		{
			name: "single resolver for colliding keys",
			schema: `type Port @key(fields: "_id") @key(fields: "id") @entityResolver(perEntity: true) {
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
				type Port @key(fields: "id") @key(fields: "i { d }") @entityResolver(perEntity: true) {
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
			schema: `type Port @key(fields: "_id") @key(fields: "id") @entityResolver(perEntity: true) {
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
				type Ship @key(fields: "owner { id }") @entityResolver(perEntity: true) { owner: Owner! }
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
				type Ship @key(fields: "id owner { id }") @entityResolver(perEntity: true, multi: true) {
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
			name: "keys sharing fields list them once",
			schema: `
				type Ship @key(fields: "id owner { id }") @key(fields: "id owner { name }")
					@entityResolver(perEntity: true) {
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
				type Boat @key(fields: "owner { name }") @entityResolver(perEntity: true) { owner: Owner! }
				type Ship @key(fields: "owner { id }") @entityResolver(perEntity: true) { owner: Owner! }
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
			name: "interface",
			schema: `
				interface Node @key(fields: "id") @entityResolver(perEntity: true) { id: ID! }
				type Dock implements Node @key(fields: "id") { id: ID! }`,
			resolver: []string{
				"findDockByID(id: ID!): Dock!",
				"findNode(rep: NodeRepresentation!): Node!",
			},
			inputs: map[string][]string{"NodeRepresentation": {"id: ID"}},
		},
		{
			name: "per-key entities are unchanged",
			schema: `
				type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! }
				type Plain @key(fields: "id") { id: ID! }`,
			resolver: []string{
				"findPlainByID(id: ID!): Plain!",
				"findPort(rep: PortRepresentation!): Port!",
			},
			inputs: map[string][]string{"PortRepresentation": {"id: ID"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cfg, err := injectFederation(t, tc.schema, tc.models)

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
		{
			name:  "per-key entity alongside",
			query: "Plain { id }",
			reps:  []map[string]any{{"__typename": "Plain", "id": "x"}},
			want:  []entity{{ID: "x"}},
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
		type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! @external }`, nil)

	require.NoError(t, err)
	require.Nil(t, cfg.Schema.Types["Entity"], "an all-external entity gets no resolver")
	require.Nil(t, cfg.Schema.Types["PortRepresentation"])
}
