//go:generate go run ../../testdata/gqlgen.go -config testdata/entityresolver/gqlgen.yml
package federation

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver"
	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver/generated"
)

func TestEntityResolver(t *testing.T) {
	srv := handler.New(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &entityresolver.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("Hello entities", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "Hello",
				"name":       "first name - 1",
			}, {
				"__typename": "Hello",
				"name":       "first name - 2",
			},
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"Hello {name}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "first name - 1", resp.Entities[0].Name)
		require.Equal(t, "first name - 2", resp.Entities[1].Name)
	})

	t.Run("HelloWithError entities", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "HelloWithErrors",
				"name":       "first name - 1",
			}, {
				"__typename": "HelloWithErrors",
				"name":       "first name - 2",
			}, {
				"__typename": "HelloWithErrors",
				"name":       "inject error",
			}, {
				"__typename": "HelloWithErrors",
				"name":       "first name - 3",
			}, {
				"__typename": "HelloWithErrors",
				"name":       "",
			},
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"HelloWithErrors {name}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.Error(t, err)
		entityErrors, err := getEntityErrors(err)
		require.NoError(t, err)
		require.Len(t, entityErrors, 2)
		errMessages := []string{
			entityErrors[0].Message,
			entityErrors[1].Message,
		}

		require.Contains(t, errMessages, "resolving Entity \"HelloWithErrors\": error (empty key) resolving HelloWithErrorsByName")
		require.Contains(t, errMessages, "resolving Entity \"HelloWithErrors\": error resolving HelloWithErrorsByName")

		require.Len(t, resp.Entities, 5)
		require.Equal(t, "first name - 1", resp.Entities[0].Name)
		require.Equal(t, "first name - 2", resp.Entities[1].Name)
		require.Equal(t, "", resp.Entities[2].Name)
		require.Equal(t, "first name - 3", resp.Entities[3].Name)
		require.Equal(t, "", resp.Entities[4].Name)
	})

	t.Run("World entities with nested key", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "World",
				"hello": map[string]any{
					"name": "world name - 1",
				},
				"foo": "foo 1",
			}, {
				"__typename": "World",
				"hello": map[string]any{
					"name": "world name - 2",
				},
				"foo": "foo 2",
			},
		}

		var resp struct {
			Entities []struct {
				Foo   string `json:"foo"`
				Hello struct {
					Name string `json:"name"`
				} `json:"hello"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"World {foo hello {name}}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "foo 1", resp.Entities[0].Foo)
		require.Equal(t, "world name - 1", resp.Entities[0].Hello.Name)
		require.Equal(t, "foo 2", resp.Entities[1].Foo)
		require.Equal(t, "world name - 2", resp.Entities[1].Hello.Name)
	})

	t.Run("World entities with multiple keys", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "WorldWithMultipleKeys",
				"hello": map[string]any{
					"name": "world name - 1",
				},
				"foo": "foo 1",
			}, {
				"__typename": "WorldWithMultipleKeys",
				"bar":        11,
			},
		}

		var resp struct {
			Entities []struct {
				Foo   string `json:"foo"`
				Hello struct {
					Name string `json:"name"`
				} `json:"hello"`
				Bar int `json:"bar"`
			} `json:"_entities"`
		}
		eq := entityQuery([]string{
			"WorldWithMultipleKeys {foo hello {name}}",
			"WorldWithMultipleKeys {bar}",
		})
		err := c.Post(
			eq,
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "foo 1", resp.Entities[0].Foo)
		require.Equal(t, "world name - 1", resp.Entities[0].Hello.Name)
		require.Equal(t, 11, resp.Entities[1].Bar)
	})

	t.Run("World entities with one single-key nil should hit resolver without nils", func(t *testing.T) {
		representations := []map[string]any{
			{
				"__typename": "WorldWithMultipleKeys",
				"hello": map[string]any{
					"name": "world name - 1",
				},
				"foo": "foo 1",
				"bar": nil,
			},
		}

		var resp struct {
			Entities []struct {
				Foo   string `json:"foo"`
				Hello struct {
					Name string `json:"name"`
				} `json:"hello"`
				Bar int `json:"bar"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"WorldWithMultipleKeys {foo hello {name} bar}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Len(t, resp.Entities, 1)
		require.Equal(t, "foo 1", resp.Entities[0].Foo)
		require.Equal(t, "world name - 1", resp.Entities[0].Hello.Name)
		require.Equal(t, entityresolver.FindWorldWithMultipleKeysByHelloNameAndFooBarValue, resp.Entities[0].Bar)
	})

	t.Run("Hello WorldName entities (heterogeneous)", func(t *testing.T) {
		// Entity resolution can handle heterogenenous representations. Meaning,
		// the representations for resolving entities can be of different
		// __typename. So the tests here will interleve two different entity
		// types so that we can test support for resolving different types and
		// correctly handle ordering.
		representations := []map[string]any{}
		count := 10

		for i := 0; i < count; i++ {
			if i%2 == 0 {
				representations = append(representations, map[string]any{
					"__typename": "Hello",
					"name":       "hello - " + strconv.Itoa(i),
				})
			} else {
				representations = append(representations, map[string]any{
					"__typename": "WorldName",
					"name":       "world name - " + strconv.Itoa(i),
				})
			}
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"Hello {name}",
				"WorldName {name}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Len(t, resp.Entities, count)

		for i := 0; i < count; i++ {
			if i%2 == 0 {
				require.Equal(t, resp.Entities[i].Name, "hello - "+strconv.Itoa(i))
			} else {
				require.Equal(t, resp.Entities[i].Name, "world name - "+strconv.Itoa(i))
			}
		}
	})

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
				Name     string `json:"name"`
				Diameter int    `json:"diameter"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"PlanetRequires {name, diameter}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "earth", resp.Entities[0].Name)
		require.Equal(t, 12, resp.Entities[0].Diameter)
		require.Equal(t, "mars", resp.Entities[1].Name)
		require.Equal(t, 10, resp.Entities[1].Diameter)
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
				Name     string `json:"name"`
				Density  int    `json:"density"`
				Diameter int    `json:"diameter"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"PlanetMultipleRequires {name, diameter, density}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)
		require.Equal(t, "earth", resp.Entities[0].Name)
		require.Equal(t, 12, resp.Entities[0].Diameter)
		require.Equal(t, 800, resp.Entities[0].Density)
		require.Equal(t, "mars", resp.Entities[1].Name)
		require.Equal(t, 10, resp.Entities[1].Diameter)
		require.Equal(t, 850, resp.Entities[1].Density)
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

func TestMultiEntityResolver(t *testing.T) {
	srv := handler.New(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &entityresolver.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("MultiHello entities", func(t *testing.T) {
		itemCount := 10
		representations := []map[string]any{}

		for i := 0; i < itemCount; i++ {
			representations = append(representations, map[string]any{
				"__typename": "MultiHello",
				"name":       "world name - " + strconv.Itoa(i),
			})
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"MultiHello {name}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)

		for i := 0; i < itemCount; i++ {
			require.Equal(t, resp.Entities[i].Name, "world name - "+strconv.Itoa(i)+" - from multiget")
		}
	})

	t.Run("MultiHello and Hello (heterogeneous) entities", func(t *testing.T) {
		itemCount := 20
		representations := []map[string]any{}

		for i := 0; i < itemCount; i++ {
			// Let's interleve the representations to test ordering of the
			// responses from the entity query
			if i%2 == 0 {
				representations = append(representations, map[string]any{
					"__typename": "MultiHello",
					"name":       "world name - " + strconv.Itoa(i),
				})
			} else {
				representations = append(representations, map[string]any{
					"__typename": "Hello",
					"name":       "hello - " + strconv.Itoa(i),
				})
			}
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"MultiHello {name}",
				"Hello {name}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.NoError(t, err)

		for i := 0; i < itemCount; i++ {
			if i%2 == 0 {
				require.Equal(t, resp.Entities[i].Name, "world name - "+strconv.Itoa(i)+" - from multiget")
			} else {
				require.Equal(t, resp.Entities[i].Name, "hello - "+strconv.Itoa(i))
			}
		}
	})

	t.Run("MultiHelloWithError entities", func(t *testing.T) {
		itemCount := 10
		representations := []map[string]any{}

		for i := 0; i < itemCount; i++ {
			representations = append(representations, map[string]any{
				"__typename": "MultiHelloWithError",
				"name":       "world name - " + strconv.Itoa(i),
			})
		}

		var resp struct {
			Entities []struct {
				Name string `json:"name"`
			} `json:"_entities"`
		}

		err := c.Post(
			entityQuery([]string{
				"MultiHelloWithError {name}",
			}),
			&resp,
			client.Var("representations", representations),
		)

		require.Error(t, err)
		entityErrors, err := getEntityErrors(err)
		require.NoError(t, err)
		require.Len(t, entityErrors, 1)
		require.Contains(t, entityErrors[0].Message, "error resolving MultiHelloWorldWithError")
	})

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

func entityQuery(queries []string) string {
	// What we want!
	// query($representations:[_Any!]!){_entities(representations:$representations){ ...on Hello{secondary} }}
	entityQueries := make([]string, len(queries))
	for i, query := range queries {
		entityQueries[i] = " ... on " + query
	}

	return "query($representations:[_Any!]!){_entities(representations:$representations){" + strings.Join(entityQueries, "") + "}}"
}

type entityResolverError struct {
	Message string   `json:"message"`
	Path    []string `json:"path"`
}

func getEntityErrors(err error) ([]*entityResolverError, error) {
	var errors []*entityResolverError
	err = json.Unmarshal([]byte(err.Error()), &errors)
	return errors, err
}
