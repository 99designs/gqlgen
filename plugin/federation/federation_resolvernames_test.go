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
	"github.com/99designs/gqlgen/plugin/federation/testdata/keycollision"
	keycollisiongen "github.com/99designs/gqlgen/plugin/federation/testdata/keycollision/generated"
)

// Not parallel: LoadSchema writes package-level caches in internal/code.
func TestEntityResolverNames(t *testing.T) {
	cases := []struct {
		name      string
		schema    string
		resolvers []string
		inputs    []string
		models    config.TypeMap // gqlgen.yml models
		err       string
	}{
		{
			name: "unique names ignore @goField",
			schema: `type Port @key(fields: "_id") @key(fields: "name") {
				_id: String! @goField(name: "LegacyID")
				name: String!
			}`,
			resolvers: []string{"findPortByID", "findPortByName"},
		},
		{
			name: "leading underscore uses @goField",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String! @goField(name: "LegacyID")
				id: ID!
			}`,
			resolvers: []string{"findPortByLegacyID", "findPortByID"},
		},
		{
			name: "leading underscore without @goField",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String!
				id: ID!
			}`,
			err: `@key(fields: "_id") and @key(fields: "id") on "Port" both generate ` +
				`entity resolver findPortByID; give one of their key fields a distinct @goField(name: ...) or gqlgen.yml fieldName, ` +
				`or, if no two key fields of one type share a Go name, set the entity_resolver_per_entity option`,
		},
		{
			name: "leading underscore uses gqlgen.yml fieldName",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String!
				id: ID!
			}`,
			models: config.TypeMap{"Port": {Fields: map[string]config.TypeMapField{
				"_id": {FieldName: "LegacyID"},
			}}},
			resolvers: []string{"findPortByLegacyID", "findPortByID"},
		},
		{
			name: "@goField beats gqlgen.yml fieldName",
			schema: `type Port @key(fields: "_id") @key(fields: "id") {
				_id: String! @goField(name: "LegacyID")
				id: ID!
			}`,
			models: config.TypeMap{"Port": {Fields: map[string]config.TypeMapField{
				"_id": {FieldName: "OldID"},
			}}},
			resolvers: []string{"findPortByLegacyID", "findPortByID"},
		},
		{
			name: "three collisions",
			schema: `type Port @key(fields: "_id") @key(fields: "id") @key(fields: "id_") {
				_id: String! @goField(name: "LegacyID")
				id: ID!
				id_: ID! @goField(name: "TrailingID")
			}`,
			resolvers: []string{"findPortByLegacyID", "findPortByID", "findPortByTrailingID"},
		},
		{
			name: "snake and camel case",
			schema: `type Port @key(fields: "port_id") @key(fields: "portID") {
				port_id: ID! @goField(name: "SnakePortID")
				portID: ID!
			}`,
			resolvers: []string{"findPortBySnakePortID", "findPortByPortID"},
		},
		{
			name: "flat and nested paths",
			schema: `
				type Port @key(fields: "id") @key(fields: "i { d }") {
					id: ID!
					i: Owner!
				}
				type Owner { d: ID! }`,
			err: `@key(fields: "id") and @key(fields: "i { d }") on "Port" both generate ` +
				`entity resolver findPortByID`,
		},
		{
			name: "flat and nested paths with @goField",
			schema: `
				type Port @key(fields: "id") @key(fields: "i { d }") {
					id: ID!
					i: Owner! @goField(name: "Inner")
				}
				type Owner { d: ID! }`,
			resolvers: []string{"findPortByID", "findPortByInnerD"},
		},
		{
			name: "flat and nested paths with gqlgen.yml fieldName on the nested type",
			schema: `
				type Port @key(fields: "id") @key(fields: "i { d }") {
					id: ID!
					i: Owner!
				}
				type Owner { d: ID! }`,
			models: config.TypeMap{"Owner": {Fields: map[string]config.TypeMapField{
				"d": {FieldName: "Code"},
			}}},
			resolvers: []string{"findPortByID", "findPortByICode"},
		},
		{
			name:   "repeated key",
			schema: `type Port @key(fields: "id") @key(fields: "id") { id: ID! }`,
			err:    `@key(fields: "id") is declared twice on "Port"`,
		},
		{
			name: "@goField name taken by another key",
			schema: `type Port @key(fields: "_id") @key(fields: "id") @key(fields: "legacyID") {
				_id: String! @goField(name: "LegacyID")
				id: ID!
				legacyID: ID!
			}`,
			err: `@key(fields: "_id") and @key(fields: "legacyID") on "Port" both generate ` +
				`entity resolver findPortByLegacyID`,
		},
		{
			name: "batch input names",
			schema: `type Port @key(fields: "_id") @key(fields: "id")
				@entityResolver(multi: true) {
				_id: String! @goField(name: "LegacyID")
				id: ID!
			}`,
			resolvers: []string{"findManyPortByLegacyIDs", "findManyPortByIDs"},
			inputs:    []string{"PortByLegacyIDsInput", "PortByIDsInput"},
		},
		{
			name: "existing type with the same Go name as an input",
			schema: `
				type Port @key(fields: "id") @entityResolver(multi: true) { id: ID! }
				input portByIDsInput { other: String }`,
			err: `type "portByIDsInput" has the same Go name as input PortByIDsInput ` +
				`of entity resolver findManyPortByIDs; rename "portByIDsInput"`,
		},
		{
			name: "different entities",
			schema: `
				type Port @key(fields: "id") @entityResolver(multi: true) { id: ID! }
				type ManyPort @key(fields: "IDs") { IDs: ID! }`,
			err: `entities "ManyPort" and "Port" both generate entity resolver findManyPortByIDs; ` +
				`giving both the same @entityResolver(multi:) setting, ` +
				`or setting the entity_resolver_per_entity option, avoids this`,
		},
		{
			name: "different entities in the same mode with the same name",
			schema: `
				type Port @key(fields: "iDByX") { iDByX: ID! }
				type PortByID @key(fields: "x") { x: ID! }`,
			err: `entities "Port" and "PortByID" both generate entity resolver findPortByIDByX; ` +
				`setting the entity_resolver_per_entity option avoids this`,
		},
		{
			name: "different entities in the same mode",
			schema: `
				type Port @key(fields: "id") @entityResolver(multi: true) { id: ID! }
				type ManyPort @key(fields: "IDs") @entityResolver(multi: true) { IDs: ID! }`,
			resolvers: []string{"findManyManyPortByIDss", "findManyPortByIDs"},
			inputs:    []string{"ManyPortByIDssInput", "PortByIDsInput"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			f, err := New(2, cfg)
			require.NoError(t, err)
			// After New, as an api.Option would: the plugin must see it.
			maps.Copy(cfg.Models, tc.models)
			early, err := f.InjectSourcesEarly()
			require.NoError(t, err)
			cfg.Sources = early
			cfg.Sources = append(cfg.Sources, &ast.Source{
				Name: "schema.graphql",
				Input: `directive @entityResolver(multi: Boolean) on OBJECT
					directive @goField(name: String) on FIELD_DEFINITION
					type Query { ok: Boolean }` + tc.schema,
			})
			require.NoError(t, cfg.LoadSchema())

			late, err := f.InjectSourcesLate(cfg.Schema)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			cfg.Sources = append(cfg.Sources, late...)
			require.NoError(t, cfg.LoadSchema())

			var resolvers, inputs []string
			for _, field := range cfg.Schema.Types["Entity"].Fields {
				if !strings.HasPrefix(field.Name, "__") {
					resolvers = append(resolvers, field.Name)
				}
			}
			for _, entity := range f.Entities {
				if !entity.Multi {
					continue
				}
				for _, resolver := range entity.Resolvers {
					inputs = append(inputs, resolver.InputTypeName)
					require.NotNil(t, cfg.Schema.Types[resolver.InputTypeName],
						"input %s missing from schema", resolver.InputTypeName)
				}
			}
			require.Equal(t, tc.resolvers, resolvers)
			require.Equal(t, tc.inputs, inputs)
		})
	}
}

// TestEntityResolverNamesRuntime resolves each key of the keycollision
// fixture's Port and Ship, checking that the router's representation reaches
// the resolver named for that key.
func TestEntityResolverNamesRuntime(t *testing.T) {
	srv := handler.New(keycollisiongen.NewExecutableSchema(keycollisiongen.Config{
		Resolvers: &keycollision.Resolver{},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	type entity struct {
		LegacyID   string `json:"_id"`
		ID         string `json:"id"`
		ResolvedBy string `json:"resolvedBy"`
	}
	cases := []struct {
		name  string
		query string
		rep   map[string]any
		want  entity
	}{
		{
			name:  "single by _id",
			query: "Port { _id id resolvedBy }",
			rep:   map[string]any{"__typename": "Port", "_id": "legacy"},
			want:  entity{LegacyID: "legacy", ResolvedBy: "_id"},
		},
		{
			name:  "single by id",
			query: "Port { _id id resolvedBy }",
			rep:   map[string]any{"__typename": "Port", "id": "p1"},
			want:  entity{ID: "p1", ResolvedBy: "id"},
		},
		{
			name:  "batch by _id",
			query: "Ship { _id id resolvedBy }",
			rep:   map[string]any{"__typename": "Ship", "_id": "legacy"},
			want:  entity{LegacyID: "legacy", ResolvedBy: "_id"},
		},
		{
			name:  "batch by id",
			query: "Ship { _id id resolvedBy }",
			rep:   map[string]any{"__typename": "Ship", "id": "s1"},
			want:  entity{ID: "s1", ResolvedBy: "id"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp struct {
				Entities []entity `json:"_entities"`
			}

			err := c.Post(entityQuery([]string{tc.query}), &resp,
				client.Var("representations", []map[string]any{tc.rep}))

			require.NoError(t, err)
			require.Equal(t, []entity{tc.want}, resp.Entities)
		})
	}
}
