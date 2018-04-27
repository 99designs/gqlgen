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
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(New())))
	c := client.New(srv.URL)

	t.Run("create a new todo", func(t *testing.T) {
		var resp struct {
			CreateTodo struct{ ID int }
		}
		c.MustPost(`mutation { createTodo(todo:{text:"Fery important"}) { id } }`, &resp)

		require.Equal(t, 4, resp.CreateTodo.ID)
	})

	t.Run("update the todo text", func(t *testing.T) {
		var resp struct {
			UpdateTodo struct{ Text string }
		}
		c.MustPost(`mutation { updateTodo(id: 4, changes:{text:"Very important"}) { text } }`, &resp)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("get __typename", func(t *testing.T) {
		var resp struct {
			Todo struct {
				Typename string `json:"__typename"`
			}
		}
		c.MustPost(`{ todo(id: 4) { __typename } }`, &resp)

		require.Equal(t, "Todo", resp.Todo.Typename)
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
			B struct{ ID int }
		}
		c.MustPost(`{ a: todo(id:1) { text } b: todo(id:2) { id } }`, &resp)

		require.Equal(t, "A todo not to forget", resp.A.Text)
		require.Equal(t, 2, resp.B.ID)
	})

	t.Run("find a missing todo", func(t *testing.T) {
		var resp struct {
			Todo *struct{ Text string }
		}
		err := c.Post(`{ todo(id:99) { text } }`, &resp)

		require.Error(t, err)
		require.Nil(t, resp.Todo)
	})

	t.Run("test panic", func(t *testing.T) {
		var resp struct {
			Todo *struct{ Text string }
		}
		err := c.Post(`{ todo(id:666) { text } }`, &resp)

		require.EqualError(t, err, `[{"message":"internal system error","path":["todo"]}]`)
	})

	t.Run("select all", func(t *testing.T) {
		var resp struct {
			Todo struct {
				ID   int
				Text string
				Done bool
			}
			LastTodo struct {
				ID   int
				Text string
				Done bool
			}
			Todos []struct {
				ID   int
				Text string
				Done bool
			}
		}
		c.MustPost(`{
			todo(id:1) { id done text }
			lastTodo { id text done }
			todos { id text done }
		}`, &resp)

		require.Equal(t, 1, resp.Todo.ID)
		require.Equal(t, 4, resp.LastTodo.ID)
		require.Len(t, resp.Todos, 4)
		require.Equal(t, "Very important", resp.LastTodo.Text)
		require.Equal(t, 4, resp.LastTodo.ID)
	})

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		var resp interface{}
		c.MustPost(introspection.Query, &resp)
	})

	t.Run("null optional field", func(t *testing.T) {
		var resp struct {
			CreateTodo struct{ Text string }
		}
		c.MustPost(`mutation { createTodo(todo:{text:"Completed todo", done: null}) { text } }`, &resp)

		require.Equal(t, "Completed todo", resp.CreateTodo.Text)
	})
}
