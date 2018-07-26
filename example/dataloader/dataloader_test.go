package dataloader

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/graphql/introspection"
	"github.com/vektah/gqlgen/handler"
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

	t.Run("customer array torture", func(t *testing.T) {
		var resp struct {
			Torture [][]Customer
		}
		c.MustPost(`{ torture(customerIds:[[1,2],[3,4,5]]) { id name } }`, &resp)

		require.EqualValues(t, [][]Customer{
			{{ID: 1, Name: "0 0"}, {ID: 2, Name: "0 1"}},
			{{ID: 3, Name: "1 0"}, {ID: 4, Name: "1 1"}, {ID: 5, Name: "1 2"}},
		}, resp.Torture)
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
		c.MustPost(`{ torture(customerIds:{}) { id name } }`, &resp)

		require.EqualValues(t, [][]Customer{}, resp.Torture)
	})

}
