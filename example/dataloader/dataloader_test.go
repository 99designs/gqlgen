package dataloader

import (
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestTodo(t *testing.T) {
	srv := httptest.NewServer(LoaderMiddleware(handler.GraphQL(NewExecutableSchema(Config{Resolvers: &Resolver{}}))))
	c := client.New(srv.URL)

	t.Run("create a new todo", func(t *testing.T) {
		var resp interface{}
		c.MustPost(`{
		  customers {
			name
			address {
			  street

			}
			orders {
			  id
              amount
			  items {
				name
			  }
			}
		  }
		}`, &resp)
	})

	t.Run("2d array marshaling", func(t *testing.T) {
		var resp struct {
			Torture2d [][]Customer
		}
		c.MustPost(`{ torture2d(customerIds:[[1,2],[3,4,5]]) { id name } }`, &resp)

		require.EqualValues(t, [][]Customer{
			{{ID: 1, Name: "0 0"}, {ID: 2, Name: "0 1"}},
			{{ID: 3, Name: "1 0"}, {ID: 4, Name: "1 1"}, {ID: 5, Name: "1 2"}},
		}, resp.Torture2d)
	})

	// Input coercion on arrays should convert non array values into an array of the appropriate depth
	// http://facebook.github.io/graphql/June2018/#sec-Type-System.List
	t.Run("array coercion", func(t *testing.T) {
		t.Run("1d", func(t *testing.T) {
			var resp struct {
				Torture1d []Customer
			}
			c.MustPost(`{ torture1d(customerIds: 1) { id name } }`, &resp)

			require.EqualValues(t, []Customer{
				{ID: 1, Name: "0"},
			}, resp.Torture1d)
		})

		t.Run("2d", func(t *testing.T) {
			var resp struct {
				Torture2d [][]Customer
			}
			c.MustPost(`{ torture2d(customerIds: 1) { id name } }`, &resp)

			require.EqualValues(t, [][]Customer{
				{{ID: 1, Name: "0 0"}},
			}, resp.Torture2d)
		})
	})

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		var resp interface{}
		c.MustPost(introspection.Query, &resp)
	})

	t.Run("customer array torture malformed array query", func(t *testing.T) {
		var resp struct {
			Torture [][]Customer
		}
		err := c.Post(`{ torture2d(customerIds:{}) { id name } }`, &resp)

		require.EqualError(t, err, "[{\"message\":\"map[string]interface {} is not an int\",\"path\":[\"torture2d\"]}]")
	})

}
