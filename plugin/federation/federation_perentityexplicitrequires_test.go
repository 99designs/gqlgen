//go:generate go run ../../testdata/gqlgen.go -config testdata/perentityexplicitrequires/gqlgen.yml
package federation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentityexplicitrequires"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentityexplicitrequires/generated"
)

// TestPerEntityExplicitRequires checks that a per-entity resolver delivers @requires through the explicit populator.
func TestPerEntityExplicitRequires(t *testing.T) {
	srv := handler.New(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &perentityexplicitrequires.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	// Diameter is not a key, so only @requires can deliver it.
	type planet struct {
		Name     string `json:"name"`
		Diameter int    `json:"diameter"`
	}
	for _, typeName := range []string{"Planet", "MultiPlanet"} {
		t.Run(typeName, func(t *testing.T) {
			var resp struct {
				Entities []planet `json:"_entities"`
			}

			err := c.Post(entityQuery([]string{typeName + " {name, diameter}"}), &resp,
				client.Var("representations", []map[string]any{
					{"__typename": typeName, "name": "earth", "diameter": 12},
					{"__typename": typeName, "name": "mars", "diameter": 10},
				}))

			require.NoError(t, err)
			require.Equal(t, []planet{{"earth", 12}, {"mars", 10}}, resp.Entities)
		})
	}
}
