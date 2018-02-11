package todo

import (
	"net/http/httptest"
	"testing"

	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/neelance/introspection"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/handler"
)

func TestTodo(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutor(New())))
	c := client.New(srv.URL)

	t.Run("create a new todo", func(t *testing.T) {
		var resp struct {
			CreateTodo struct{ ID string }
		}
		c.MustPost(`mutation { createTodo(text:"Fery important") { id } }`, &resp)

		require.Equal(t, "t4", resp.CreateTodo.ID)
	})

	t.Run("update the todo text", func(t *testing.T) {
		var resp struct {
			UpdateTodo struct{ Text string }
		}
		c.MustPost(`mutation { updateTodo(id: 4, changes:{text:"Very important"}) { text } }`, &resp)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("update the todo status", func(t *testing.T) {
		var resp struct {
			UpdateTodo struct{ Text string }
		}
		c.MustPost(`mutation { updateTodo(id: 4, changes:{done:true}) { text } }`, &resp)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("select with alias", func(t *testing.T) {
		var resp struct {
			A struct{ Text string }
			B struct{ ID string }
		}
		c.MustPost(`{ a: todo(id:1) { text } b: todo(id:2) { id } }`, &resp)

		require.Equal(t, "A todo not to forget", resp.A.Text)
		require.Equal(t, "t2", resp.B.ID)
	})

	t.Run("find a missing todo", func(t *testing.T) {
		var resp struct {
			Todo *struct{ Text string }
		}
		err := c.Post(`{ todo(id:99) { text } }`, &resp)

		require.Error(t, err)
		require.Nil(t, resp.Todo)
	})

	t.Run("select all", func(t *testing.T) {
		var resp struct {
			Todo struct {
				ID   string
				Text string
				Done bool
			}
			LastTodo struct {
				ID   string
				Text string
				Done bool
			}
			Todos []struct {
				ID   string
				Text string
				Done bool
			}
		}
		c.MustPost(`{
			todo(id:1) { id done text }
			lastTodo { id text done }
			todos { id text done }
		}`, &resp)

		require.Equal(t, "t1", resp.Todo.ID)
		require.Equal(t, "t4", resp.LastTodo.ID)
		require.Len(t, resp.Todos, 4)
		require.Equal(t, "Very important", resp.LastTodo.Text)
		require.Equal(t, "t4", resp.LastTodo.ID)
	})

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		c.MustPost(introspection.Query, nil)
	})
}
