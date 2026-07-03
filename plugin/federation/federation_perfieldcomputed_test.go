//go:generate go run ../../testdata/gqlgen.go -config testdata/perfieldcomputed/gqlgen.yml
package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perfieldcomputed"
	perfieldcomputedgen "github.com/99designs/gqlgen/plugin/federation/testdata/perfieldcomputed/generated"
)

// TestPerFieldComputedRequires proves @computedRequires makes the computed
// strategy per field: on one preloaded entity, the scalar @requires (category) is
// populated onto the batch resolver's input while the object @requires (info) is
// delivered to a standalone field resolver — the mix preloaded alone cannot
// express. Both resolve in one _entities request.
func TestPerFieldComputedRequires(t *testing.T) {
	srv := handler.New(
		perfieldcomputedgen.NewExecutableSchema(perfieldcomputedgen.Config{
			Resolvers: &perfieldcomputed.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	representations := []map[string]any{
		{
			"__typename": "Product",
			"id":         "1",
			"category":   "books",
			"info":       map[string]any{"label": "hardcover"},
		},
		{
			"__typename": "Product",
			"id":         "2",
			"category":   "toys",
			"info":       map[string]any{"label": "wooden"},
		},
	}

	var resp struct {
		Entities []struct {
			ID      string `json:"id"`
			Display string `json:"display"`
			Summary string `json:"summary"`
		} `json:"_entities"`
	}

	err := c.Post(
		entityQuery([]string{"Product { id display summary }"}),
		&resp,
		client.Var("representations", representations),
	)
	require.NoError(t, err)
	require.Len(t, resp.Entities, 2)

	// display: preloaded scalar @requires (category) reached the batch resolver input.
	assert.Equal(t, "1", resp.Entities[0].ID)
	assert.Equal(t, "display for books", resp.Entities[0].Display)
	// summary: object @requires (info) reached the @computedRequires field resolver.
	assert.Equal(t, "summary: hardcover", resp.Entities[0].Summary)

	assert.Equal(t, "2", resp.Entities[1].ID)
	assert.Equal(t, "display for toys", resp.Entities[1].Display)
	assert.Equal(t, "summary: wooden", resp.Entities[1].Summary)
}
