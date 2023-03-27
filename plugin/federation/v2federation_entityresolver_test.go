//go:generate go run ../../testdata/gqlgen.go -config testdata/federatedentityresolver/gqlgen.yml
package federation

// import (
// 	"strconv"
// 	"testing"

// 	"github.com/99designs/gqlgen/client"
// 	"github.com/99designs/gqlgen/graphql/handler"
// 	"github.com/stretchr/testify/require"

// 	"github.com/99designs/gqlgen/plugin/federation/testdata/federatedentityresolver/generated"
// 	"github.com/99designs/gqlgen/plugin/federation/testdata/federatedentityresolver/resolvers"
// )

// func TestFederatedEntityResolver(t *testing.T) {
// 	c := client.New(handler.NewDefaultServer(
// 		generated.NewExecutableSchema(generated.Config{
// 			Resolvers: &resolvers.Resolver{},
// 		}),
// 	))

// 	t.Run("Hello entities", func(t *testing.T) {
// 		representations := []map[string]interface{}{
// 			{
// 				"__typename": "Hello",
// 				"name":       "first name - 1",
// 			}, {
// 				"__typename": "Hello",
// 				"name":       "first name - 2",
// 			},
// 		}

// 		var resp struct {
// 			Entities []struct {
// 				Name string `json:"name"`
// 			} `json:"_entities"`
// 		}

// 		err := c.Post(
// 			entityQuery([]string{
// 				"Hello {name}",
// 			}),
// 			&resp,
// 			client.Var("representations", representations),
// 		)

// 		require.NoError(t, err)
// 		require.Equal(t, resp.Entities[0].Name, "first name - 1")
// 		require.Equal(t, resp.Entities[1].Name, "first name - 2")
// 	})
// }

// func TestFederatedMultiEntityResolver(t *testing.T) {
// 	c := client.New(handler.NewDefaultServer(
// 		generated.NewExecutableSchema(generated.Config{
// 			Resolvers: &resolvers.Resolver{},
// 		}),
// 	))

// 	t.Run("MultiHello entities", func(t *testing.T) {
// 		itemCount := 10
// 		representations := []map[string]interface{}{}

// 		for i := 0; i < itemCount; i++ {
// 			representations = append(representations, map[string]interface{}{
// 				"__typename": "MultiHello",
// 				"name":       "world name - " + strconv.Itoa(i),
// 			})
// 		}

// 		var resp struct {
// 			Entities []struct {
// 				Name string `json:"name"`
// 			} `json:"_entities"`
// 		}

// 		err := c.Post(
// 			entityQuery([]string{
// 				"MultiHello {name}",
// 			}),
// 			&resp,
// 			client.Var("representations", representations),
// 		)

// 		require.NoError(t, err)

// 		for i := 0; i < itemCount; i++ {
// 			require.Equal(t, resp.Entities[i].Name, "world name - "+strconv.Itoa(i)+" - from multiget")
// 		}
// 	})
// }
