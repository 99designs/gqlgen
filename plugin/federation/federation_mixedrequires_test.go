//go:generate go run ../../testdata/gqlgen.go -config testdata/mixedrequires/gqlgen.yml
package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/mixedrequires"
	mixedgen "github.com/99designs/gqlgen/plugin/federation/testdata/mixedrequires/generated"
)

// TestMixedRequiresStrategies proves the package-global limitation is gone: in a
// single subgraph, Planet uses the `computed` @requires strategy and Product
// uses `preloaded`, both selected per entity via @entityResolver and
// with no package-level option. One _entities request resolves both.
func TestMixedRequiresStrategies(t *testing.T) {
	srv := handler.New(
		mixedgen.NewExecutableSchema(mixedgen.Config{
			Resolvers: &mixedrequires.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	representations := []map[string]any{
		{"__typename": "Planet", "name": "earth", "diameter": 12742},
		{"__typename": "Product", "id": "1", "category": "books"},
		{"__typename": "Planet", "name": "mars", "diameter": 6779},
	}

	var resp struct {
		Entities []struct {
			// Planet fields
			Name string `json:"name"`
			Size int    `json:"size"`
			// Product fields
			ID      string `json:"id"`
			Display string `json:"display"`
		} `json:"_entities"`
	}

	err := c.Post(
		entityQuery([]string{
			"Planet { name size }",
			"Product { id display }",
		}),
		&resp,
		client.Var("representations", representations),
	)
	require.NoError(t, err)
	require.Len(t, resp.Entities, 3)

	// Planet: `size` came from the computed field resolver reading @requires diameter.
	assert.Equal(t, "earth", resp.Entities[0].Name)
	assert.Equal(t, 12742, resp.Entities[0].Size)
	// Product: `display` came from the preloaded batch resolver reading
	// @requires category off the populated input.
	assert.Equal(t, "1", resp.Entities[1].ID)
	assert.Equal(t, "display for books", resp.Entities[1].Display)
	// Planet again, proving the computed path handles the batch by position.
	assert.Equal(t, "mars", resp.Entities[2].Name)
	assert.Equal(t, 6779, resp.Entities[2].Size)
}
