package todo

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/graphql-go"
	"github.com/vektah/graphql-go/client"
	"github.com/vektah/graphql-go/introspection"
)

func TestTodo(t *testing.T) {
	srv := httptest.NewServer(graphql.Handler(NewExecutor(New())))
	c := client.New(srv.URL)

	t.Run("create a new todo", func(t *testing.T) {
		var resp struct {
			CreateTodo Todo
		}
		c.MustPost(`mutation { createTodo(text:"Fery important") { id } }`, &resp)

		require.Equal(t, 4, resp.CreateTodo.ID)
	})

	t.Run("update the todo text", func(t *testing.T) {
		var resp struct {
			UpdateTodo Todo
		}
		c.MustPost(`mutation { updateTodo(id: 4, changes:{text:"Very important"}) { text } }`, &resp)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("update the todo status", func(t *testing.T) {
		var resp struct {
			UpdateTodo Todo
		}
		c.MustPost(`mutation { updateTodo(id: 4, changes:{done:true}) { text } }`, &resp)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("select all", func(t *testing.T) {
		var resp struct {
			Todo     Todo
			LastTodo Todo
			Todos    []Todo
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
		c.MustPost(introspection.Query, nil)
	})
}
