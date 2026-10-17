//go:generate go run ../../testdata/gqlgen.go -config testdata/perentity/gqlgen.yml
package federation

import (
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

// injectFederation loads schema with the federation sources injected, as
// api.Generate does, and returns the error from the late injection, which is
// the step under test. Not parallel-safe: LoadSchema writes package-level
// caches in internal/code.
func injectFederation(t *testing.T, schema string) (*Federation, *ast.Schema, error) {
	t.Helper()

	f := &Federation{version: 2}
	early, err := f.InjectSourcesEarly()
	require.NoError(t, err)
	cfg := config.DefaultConfig()
	cfg.Sources = early
	cfg.Sources = append(cfg.Sources, &ast.Source{
		Name: "schema.graphql",
		Input: `directive @entityResolver(multi: Boolean, requires: String, perEntity: Boolean) on OBJECT
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
	return f, cfg.Schema, nil
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
			f, _, err := injectFederation(t, tc.schema)

			require.NoError(t, err)
			require.Len(t, f.Entities, 1)
			require.Equal(t, tc.want, f.Entities[0].PerEntity)
		})
	}
}

func TestPerEntityRejectsPreloaded(t *testing.T) {
	_, _, err := injectFederation(t, `
		type Port @key(fields: "id")
			@entityResolver(perEntity: true, multi: true, requires: "preloaded") {
			id: ID!
		}`)

	require.ErrorContains(
		t,
		err,
		`entity "Port": @entityResolver(perEntity: true) cannot be combined`,
	)
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
		resolver []string
		inputs   map[string][]string
	}{
		{
			name: "single resolver for colliding keys",
			schema: `type Port @key(fields: "_id") @key(fields: "id") @entityResolver(perEntity: true) {
				_id: String!
				id: ID!
			}`,
			resolver: []string{"findPort(rep: PortRepresentation!): Port!"},
			inputs:   map[string][]string{"PortRepresentation": {"_id: String", "id: ID"}},
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
			_, schema, err := injectFederation(t, tc.schema)

			require.NoError(t, err)
			require.Equal(t, tc.resolver, fieldsSDL(schema.Types["Entity"]))
			for name, fields := range tc.inputs {
				require.NotNil(t, schema.Types[name], "input %s missing from schema", name)
				require.Equal(t, ast.InputObject, schema.Types[name].Kind)
				require.Equal(t, fields, fieldsSDL(schema.Types[name]), "fields of %s", name)
			}
		})
	}
}

func TestPerEntityRejectsExistingRepresentationName(t *testing.T) {
	_, _, err := injectFederation(t, `
		type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! }
		input PortRepresentation { other: String }`)

	require.ErrorContains(t, err, `entity "Port": type PortRepresentation already exists`)
}

func TestCopyRepresentationFieldNames(t *testing.T) {
	models := config.TypeMap{
		"Port": {Fields: map[string]config.TypeMapField{
			"_id": {FieldName: "LegacyID"},
		}},
		"Owner": {Fields: map[string]config.TypeMapField{
			"id": {FieldName: "OwnerKey"},
		}},
		"OwnerRepresentation": {Fields: map[string]config.TypeMapField{
			"id": {FieldName: "ChosenByUser"},
		}},
	}
	representations := []*representation{
		{Name: "PortRepresentation", Owner: "Port", Fields: []*representationField{
			{Name: "_id", Type: "String"},
			{Name: "id", Type: "ID"},
		}},
		{Name: "OwnerRepresentation", Owner: "Owner", Fields: []*representationField{
			{Name: "id", Type: "ID"},
		}},
	}

	copyRepresentationFieldNames(representations, models)

	require.Equal(t, map[string]config.TypeMapField{
		"_id": {FieldName: "LegacyID"},
	}, models["PortRepresentation"].Fields, "copies the owner's name; leaves unnamed fields alone")
	require.Equal(t, "ChosenByUser", models["OwnerRepresentation"].Fields["id"].FieldName,
		"keeps a name the user set on the representation")
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

	t.Run("representation without a key", func(t *testing.T) {
		var resp map[string]any

		err := c.Post(entityQuery([]string{"Port { id }"}), &resp,
			client.Var("representations", []map[string]any{{"__typename": "Port", "size": 1}}))

		require.ErrorContains(t, err, `missing Key Field \"_id\" for Port`)
	})
}

func TestPerEntityIgnoredWithoutResolvers(t *testing.T) {
	_, schema, err := injectFederation(t, `
		type Port @key(fields: "id") @entityResolver(perEntity: true) { id: ID! @external }`)

	require.NoError(t, err)
	require.Nil(t, schema.Types["Entity"], "an all-external entity gets no resolver")
	require.Nil(t, schema.Types["PortRepresentation"])
}
