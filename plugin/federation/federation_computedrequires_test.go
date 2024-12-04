//go:generate go run ../../testdata/gqlgen.go -config testdata/computedrequires/gqlgen.yml
package federation

import (
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/plugin/federation/testdata/computedrequires"
	"github.com/99designs/gqlgen/plugin/federation/testdata/computedrequires/generated"
)

func TestComputedRequires(t *testing.T) {
	srv := handler.New(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &computedrequires.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("PlanetRequires entities with requires directive", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "PlanetRequires",
				"name":       "earth",
				"diameter":   12,
			}, {
				"__typename": "PlanetRequires",
				"name":       "mars",
				"diameter":   10,
			},
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
				Size int    `json:"size"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"PlanetRequires {name size}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "earth", resp.Entities[0].Name)
		require.Equal(t, 12, resp.Entities[0].Size)
		require.Equal(t, "mars", resp.Entities[1].Name)
		require.Equal(t, 10, resp.Entities[1].Size)
	})

	t.Run("PlanetRequires entities with multiple required fields directive", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "PlanetMultipleRequires",
				"name":       "earth",
				"density":    800,
				"diameter":   12,
			}, {
				"__typename": "PlanetMultipleRequires",
				"name":       "mars",
				"density":    850,
				"diameter":   10,
			},
		}

		var resp struct {
			Entities []struct {
				Name   string `json:"name"`
				Weight int    `json:"weight"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"PlanetMultipleRequires {name weight}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "earth", resp.Entities[0].Name)
		require.Equal(t, 812, resp.Entities[0].Weight)
		require.Equal(t, "mars", resp.Entities[1].Name)
		require.Equal(t, 860, resp.Entities[1].Weight)
	})

	t.Run("PlanetRequiresNested entities with requires directive having nested field", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "PlanetRequiresNested",
				"name":       "earth",
				"world": map[string]any{
					"foo": "A",
				},
			}, {
				"__typename": "PlanetRequiresNested",
				"name":       "mars",
				"world": map[string]any{
					"foo": "B",
				},
			},
		}

		var resp struct {
			Entities []struct {
				Name  string `json:"name"`
				World struct {
					Foo string `json:"foo"`
				} `json:"world"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"PlanetRequiresNested {name, world { foo }}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "earth", resp.Entities[0].Name)
		require.Equal(t, "A", resp.Entities[0].World.Foo)
		require.Equal(t, "mars", resp.Entities[1].Name)
		require.Equal(t, "B", resp.Entities[1].World.Foo)
	})
}

func TestMultiComputedRequires(t *testing.T) {
	srv := handler.New(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &computedrequires.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("MultiHelloRequires entities with requires directive", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "MultiHelloRequires",
				"name":       "first name - 1",
				"key1":       "key1 - 1",
			}, {
				"__typename": "MultiHelloRequires",
				"name":       "first name - 2",
				"key1":       "key1 - 2",
			},
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
				Key1 string `json:"key1"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"MultiHelloRequires {name, key1}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "first name - 1", resp.Entities[0].Name)
		require.Equal(t, "key1 - 1", resp.Entities[0].Key1)
		require.Equal(t, "first name - 2", resp.Entities[1].Name)
		require.Equal(t, "key1 - 2", resp.Entities[1].Key1)
	})

	t.Run("MultiHelloMultipleRequires entities with multiple required fields", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "MultiHelloMultipleRequires",
				"name":       "first name - 1",
				"key1":       "key1 - 1",
				"key2":       "key2 - 1",
			}, {
				"__typename": "MultiHelloMultipleRequires",
				"name":       "first name - 2",
				"key1":       "key1 - 2",
				"key2":       "key2 - 2",
			},
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
				Key1 string `json:"key1"`
				Key2 string `json:"key2"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"MultiHelloMultipleRequires {name, key1, key2}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "first name - 1", resp.Entities[0].Name)
		require.Equal(t, "key1 - 1", resp.Entities[0].Key1)
		require.Equal(t, "key2 - 1", resp.Entities[0].Key2)
		require.Equal(t, "first name - 2", resp.Entities[1].Name)
		require.Equal(t, "key1 - 2", resp.Entities[1].Key1)
		require.Equal(t, "key2 - 2", resp.Entities[1].Key2)
	})

	t.Run("MultiPlanetRequiresNested entities with requires directive having nested field", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "MultiPlanetRequiresNested",
				"name":       "earth",
				"world": map[string]any{
					"foo": "A",
				},
			}, {
				"__typename": "MultiPlanetRequiresNested",
				"name":       "mars",
				"world": map[string]any{
					"foo": "B",
				},
			},
		}

		var resp struct {
			Entities []struct {
				Name  string `json:"name"`
				World struct {
					Foo string `json:"foo"`
				} `json:"world"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"MultiPlanetRequiresNested {name, world { foo }}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "earth", resp.Entities[0].Name)
		require.Equal(t, "A", resp.Entities[0].World.Foo)
		require.Equal(t, "mars", resp.Entities[1].Name)
		require.Equal(t, "B", resp.Entities[1].World.Foo)
	})
}
