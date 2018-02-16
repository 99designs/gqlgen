package dataloader

import (
	"net/http/httptest"
	"testing"

	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/handler"
	"github.com/vektah/gqlgen/neelance/introspection"
)

func TestTodo(t *testing.T) {
	srv := httptest.NewServer(LoaderMiddleware(handler.GraphQL(NewExecutor(&Resolver{}))))
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

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		var resp interface{}
		c.MustPost(introspection.Query, &resp)
	})
}
