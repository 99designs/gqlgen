//go:generate go run ../../testdata/gqlgen.go -config testdata/perentitypreloaded/gqlgen.yml
package federation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentitypreloaded"
	"github.com/99designs/gqlgen/plugin/federation/testdata/perentitypreloaded/generated"
)

func TestPerEntityPreloaded(t *testing.T) {
	srv := handler.New(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &perentitypreloaded.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	// The resolvers compute size from the representation's diameter and
	// leave diameter unset: preloaded does not copy @requires onto the entity.
	type planet struct {
		Name     string `json:"name"`
		Diameter int    `json:"diameter"`
		Size     int    `json:"size"`
	}
	for _, typeName := range []string{"Planet", "MultiPlanet"} {
		t.Run(typeName, func(t *testing.T) {
			var resp struct {
				Entities []planet `json:"_entities"`
			}

			err := c.Post(entityQuery([]string{typeName + " {name, diameter, size}"}), &resp,
				client.Var("representations", []map[string]any{
					{"__typename": typeName, "name": "earth", "diameter": 12},
					{"__typename": typeName, "name": "mars", "diameter": 7},
				}))

			require.NoError(t, err)
			require.Equal(t, []planet{{"earth", 0, 24}, {"mars", 0, 14}}, resp.Entities)
		})
	}

	t.Run("nested @requires, sent or not", func(t *testing.T) {
		type comet struct {
			Name   string `json:"name"`
			Period *int   `json:"period"`
		}
		var resp struct {
			Entities []comet `json:"_entities"`
		}

		err := c.Post(entityQuery([]string{"Comet {name, period}"}), &resp,
			client.Var("representations", []map[string]any{
				{"__typename": "Comet", "name": "halley", "orbit": map[string]any{"period": 76}},
				{"__typename": "Comet", "name": "unknown"},
			}))

		require.NoError(t, err)
		period := 76
		require.Equal(t, []comet{{"halley", &period}, {"unknown", nil}}, resp.Entities)
	})
}
