package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/keycollision"
	keycollisiongen "github.com/99designs/gqlgen/plugin/federation/testdata/keycollision/generated"
)

// TestKeyFieldCollisionDisambiguated is the regression test for the key
// collision bug: @key(fields: "id i { d }") has two key paths that both reduce
// to the Go name "ID". They are now disambiguated to ID and ID2 on the generated
// multi-resolver input type, so the schema loads cleanly instead of failing with
// a duplicate-field error.
func TestKeyFieldCollisionDisambiguated(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/keycollision/keycollision.yml")
	require.NoError(t, err)
	if cfg.Federation.Version == 0 {
		cfg.Federation.Version = 1
	}

	f := &Federation{version: cfg.Federation.Version}

	early, err := f.InjectSourcesEarly()
	require.NoError(t, err)
	cfg.Sources = append(cfg.Sources, early...)
	require.NoError(t, cfg.LoadSchema())

	late, err := f.InjectSourcesLate(cfg.Schema)
	require.NoError(t, err)
	require.Len(t, f.Entities, 1)

	resolver := f.Entities[0].Resolvers[0]
	require.Len(t, resolver.KeyFields, 2)

	// Both key paths reduce to "ID"; the second is disambiguated to "ID2".
	assert.Equal(t, "ID", resolver.KeyFields[0].Field.ToGo())
	assert.Equal(t, "ID", resolver.KeyFields[1].Field.ToGo())
	assert.Equal(t, "ID", resolver.KeyFields[0].GoName)
	assert.Equal(t, "ID2", resolver.KeyFields[1].GoName)

	// The generated input type carries two distinct fields, not a duplicate.
	inputSDL := buildEntityResolverInputDefinitionSDL(resolver)
	assert.Contains(t, inputSDL, "ID: ID!")
	assert.Contains(t, inputSDL, "ID2: String!")

	// The full schema (including the late-injected input type) now loads.
	cfg.Sources = append(cfg.Sources, late...)
	require.NoError(t, cfg.LoadSchema())
}

// TestKeyFieldCollisionRuntime resolves a Collision entity by both colliding key
// paths end to end, proving each disambiguated field (ID from "id", ID2 from
// "i { d }") is independently readable in the resolver.
func TestKeyFieldCollisionRuntime(t *testing.T) {
	srv := handler.New(
		keycollisiongen.NewExecutableSchema(keycollisiongen.Config{
			Resolvers: &keycollision.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	representations := []map[string]any{
		{"__typename": "Collision", "id": "1", "i": map[string]any{"d": "x"}},
		{"__typename": "Collision", "id": "2", "i": map[string]any{"d": "y"}},
	}

	var resp struct {
		Entities []struct {
			ID string `json:"id"`
			I  struct {
				D string `json:"d"`
			} `json:"i"`
		} `json:"_entities"`
	}

	err := c.Post(
		entityQuery([]string{"Collision { id i { d } }"}),
		&resp,
		client.Var("representations", representations),
	)
	require.NoError(t, err)
	require.Len(t, resp.Entities, 2)

	// id -> ID, i.d -> ID2; the resolver echoed both back.
	assert.Equal(t, "1", resp.Entities[0].ID)
	assert.Equal(t, "x", resp.Entities[0].I.D)
	assert.Equal(t, "2", resp.Entities[1].ID)
	assert.Equal(t, "y", resp.Entities[1].I.D)
}
