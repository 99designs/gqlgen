package federation

import (
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver"
	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolver/generated"
)

func TestEntityResolver(t *testing.T) {
	c := client.New(handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: &entityresolver.Resolver{}}),
	))

	t.Run("Hello entities - single federation key", func(t *testing.T) {
		representations := []map[string]interface{}{
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
		require.Equal(t, resp.Entities[0].Name, "first name - 1")
		require.Equal(t, resp.Entities[1].Name, "first name - 2")
	})

	t.Run("World entity with nested key", func(t *testing.T) {
		representations := []map[string]interface{}{
			{
				"__typename": "World",
				"hello": map[string]interface{}{
					"name": "world name - 1",
				},
				"foo": "foo 1",
			}, {
				"__typename": "World",
				"hello": map[string]interface{}{
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
		require.Equal(t, resp.Entities[0].Foo, "foo 1")
		require.Equal(t, resp.Entities[0].Hello.Name, "world name - 1")
		require.Equal(t, resp.Entities[1].Foo, "foo 2")
		require.Equal(t, resp.Entities[1].Hello.Name, "world name - 2")
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
