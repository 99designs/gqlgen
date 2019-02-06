package todo

import (
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestTodo(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(New())))
	c := client.New(srv.URL)

	var resp struct {
		CreateTodo struct{ ID string }
	}
	c.MustPost(`mutation { createTodo(todo:{text:"Fery important"}) { id } }`, &resp)

	require.Equal(t, "5", resp.CreateTodo.ID)

	t.Run("update the todo text", func(t *testing.T) {
		var resp struct {
			UpdateTodo struct{ Text string }
		}
		c.MustPost(
			`mutation($id: ID!, $text: String!) { updateTodo(id: $id, changes:{text:$text}) { text } }`,
			&resp,
			client.Var("id", 5),
			client.Var("text", "Very important"),
		)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("get __typename", func(t *testing.T) {
		var resp struct {
			Todo struct {
				Typename string `json:"__typename"`
			}
		}
		c.MustPost(`{ todo(id: 5) { __typename } }`, &resp)

		require.Equal(t, "Todo", resp.Todo.Typename)
	})

	t.Run("update the todo status", func(t *testing.T) {
		var resp struct {
			UpdateTodo struct{ Text string }
		}
		c.MustPost(`mutation { updateTodo(id: 5, changes:{done:true}) { text } }`, &resp)

		require.Equal(t, "Very important", resp.UpdateTodo.Text)
	})

	t.Run("select with alias", func(t *testing.T) {
		var resp struct {
			A struct{ Text string }
			B struct{ ID string }
		}
		c.MustPost(`{ a: todo(id:1) { text } b: todo(id:2) { id } }`, &resp)

		require.Equal(t, "A todo not to forget", resp.A.Text)
		require.Equal(t, "2", resp.B.ID)
	})

	t.Run("find a missing todo", func(t *testing.T) {
		var resp struct {
			Todo *struct{ Text string }
		}
		err := c.Post(`{ todo(id:99) { text } }`, &resp)

		require.Error(t, err)
		require.Nil(t, resp.Todo)
	})

	t.Run("get done status of unowned todo", func(t *testing.T) {
		var resp struct {
			Todo *struct {
				Text string
				Done bool
			}
		}
		err := c.Post(`{ todo(id:3) { text, done } }`, &resp)

		require.EqualError(t, err, `[{"message":"you dont own that","path":["todo","done"]}]`)
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
			}
		}
		c.MustPost(`{
			todo(id:1) { id done text }
			lastTodo { id text done }
			todos { id text }
		}`, &resp)

		require.Equal(t, "1", resp.Todo.ID)
		require.Equal(t, "5", resp.LastTodo.ID)
		require.Len(t, resp.Todos, 5)
		require.Equal(t, "Very important", resp.LastTodo.Text)
		require.Equal(t, "5", resp.LastTodo.ID)
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

func TestSkipAndIncludeDirectives(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(New())))
	c := client.New(srv.URL)

	t.Run("skip on field", func(t *testing.T) {
		var resp map[string]interface{}
		c.MustPost(`{ todo(id: 1) @skip(if:true) { __typename } }`, &resp)
		_, ok := resp["todo"]
		require.False(t, ok)
	})

	t.Run("skip on variable", func(t *testing.T) {
		q := `query Test($cond: Boolean!) { todo(id: 1) @skip(if: $cond) { __typename } }`
		var resp map[string]interface{}

		c.MustPost(q, &resp, client.Var("cond", true))
		_, ok := resp["todo"]
		require.False(t, ok)

		c.MustPost(q, &resp, client.Var("cond", false))
		_, ok = resp["todo"]
		require.True(t, ok)
	})

	t.Run("skip on inline fragment", func(t *testing.T) {
		var resp struct {
			Todo struct {
				Typename string `json:"__typename"`
			}
		}
		c.MustPost(`{ todo(id: 1) {
				...@skip(if:true) {
					__typename
				}
			}
		}`, &resp)
		require.Empty(t, resp.Todo.Typename)
	})

	t.Run("skip on fragment", func(t *testing.T) {
		var resp struct {
			Todo struct {
				Typename string `json:"__typename"`
			}
		}
		c.MustPost(`
		{
			todo(id: 1) {
				...todoFragment @skip(if:true)
			}
		}
		fragment todoFragment on Todo {
			__typename
		}
		`, &resp)
		require.Empty(t, resp.Todo.Typename)
	})

	t.Run("include on field", func(t *testing.T) {
		q := `query Test($cond: Boolean!) { todo(id: 1) @include(if: $cond) { __typename } }`
		var resp map[string]interface{}

		c.MustPost(q, &resp, client.Var("cond", true))
		_, ok := resp["todo"]
		require.True(t, ok)

		c.MustPost(q, &resp, client.Var("cond", false))
		_, ok = resp["todo"]
		require.False(t, ok)
	})

	t.Run("both skip and include defined", func(t *testing.T) {
		type TestCase struct {
			Skip     bool
			Include  bool
			Expected bool
		}
		table := []TestCase{
			{Skip: true, Include: true, Expected: false},
			{Skip: true, Include: false, Expected: false},
			{Skip: false, Include: true, Expected: true},
			{Skip: false, Include: false, Expected: false},
		}
		q := `query Test($skip: Boolean!, $include: Boolean!) { todo(id: 1) @skip(if: $skip) @include(if: $include) { __typename } }`
		for _, tc := range table {
			var resp map[string]interface{}
			c.MustPost(q, &resp, client.Var("skip", tc.Skip), client.Var("include", tc.Include))
			_, ok := resp["todo"]
			require.Equal(t, tc.Expected, ok)
		}
	})

	t.Run("skip with default query argument", func(t *testing.T) {
		var resp map[string]interface{}
		c.MustPost(`query Test($skip: Boolean = true) { todo(id: 1) @skip(if: $skip) { __typename } }`, &resp)
		_, ok := resp["todo"]
		require.False(t, ok)
	})
}
