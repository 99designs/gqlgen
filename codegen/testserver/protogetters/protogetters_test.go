//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go
package protogetters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/codegen/testserver/protogetters/models"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestProtoGetters(t *testing.T) {
	resolver := &Resolver{}
	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver})))

	t.Run("user with all fields set", func(t *testing.T) {
		name := "Alice"
		email := "alice@example.com"
		resolver.users = map[string]*models.User{
			"1": models.NewUser("1", &name, &email, 30),
		}

		var resp struct {
			User struct {
				ID    string
				Name  *string
				Email *string
				Age   int
			}
		}
		c.MustPost(`{user(id:"1"){id name email age}}`, &resp)

		require.Equal(t, "1", resp.User.ID)
		require.NotNil(t, resp.User.Name)
		require.Equal(t, "Alice", *resp.User.Name)
		require.NotNil(t, resp.User.Email)
		require.Equal(t, "alice@example.com", *resp.User.Email)
		require.Equal(t, 30, resp.User.Age)
	})

	t.Run("user with nullable fields unset", func(t *testing.T) {
		resolver.users = map[string]*models.User{
			"2": models.NewUser("2", nil, nil, 25),
		}

		var resp struct {
			User struct {
				ID    string
				Name  *string
				Email *string
				Age   int
			}
		}
		c.MustPost(`{user(id:"2"){id name email age}}`, &resp)

		require.Equal(t, "2", resp.User.ID)
		require.Nil(t, resp.User.Name)
		require.Nil(t, resp.User.Email)
		require.Equal(t, 25, resp.User.Age)
	})

	t.Run("user with only name set", func(t *testing.T) {
		name := "Bob"
		resolver.users = map[string]*models.User{
			"3": models.NewUser("3", &name, nil, 40),
		}

		var resp struct {
			User struct {
				ID    string
				Name  *string
				Email *string
				Age   int
			}
		}
		c.MustPost(`{user(id:"3"){id name email age}}`, &resp)

		require.Equal(t, "3", resp.User.ID)
		require.NotNil(t, resp.User.Name)
		require.Equal(t, "Bob", *resp.User.Name)
		require.Nil(t, resp.User.Email)
		require.Equal(t, 40, resp.User.Age)
	})
}

func TestProtoInputSetters(t *testing.T) {
	cfg := Config{Resolvers: &Resolver{}}
	cfg.Directives.FilterEmail = func(ctx context.Context, _ any, next graphql.Resolver) (any, error) {
		value, err := next(ctx)
		if email, ok := value.(*string); ok && email != nil && *email == "discard" {
			return nil, nil
		}
		return value, err
	}
	c := client.New(handler.NewDefaultServer(NewExecutableSchema(cfg)))

	run := func(t *testing.T, input string, wantName, wantEmail *string, wantAge int) {
		t.Helper()
		var resp struct {
			UpdateUser struct {
				ID    string
				Name  *string
				Email *string
				Age   int
			}
		}
		c.MustPost(`mutation { updateUser(input: `+input+`) { id name email age } }`, &resp)
		require.Equal(t, "1", resp.UpdateUser.ID)
		require.Equal(t, wantName, resp.UpdateUser.Name)
		require.Equal(t, wantEmail, resp.UpdateUser.Email)
		require.Equal(t, wantAge, resp.UpdateUser.Age)
	}

	t.Run("omitted fields and default", func(t *testing.T) {
		run(t, `{id: "1"}`, nil, nil, 18)
	})
	t.Run("explicit null", func(t *testing.T) {
		run(t, `{id: "1", name: null, email: null}`, nil, nil, 18)
	})
	t.Run("zero values are present", func(t *testing.T) {
		empty := ""
		run(t, `{id: "1", name: "", email: "", age: 0}`, &empty, &empty, 0)
	})
	t.Run("directive returns null", func(t *testing.T) {
		name := "Alice"
		run(t, `{id: "1", name: "Alice", email: "discard"}`, &name, nil, 18)
	})
}
