package testserver

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestTime(t *testing.T) {
	resolvers := &Stub{}

	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolvers})))
	c := client.New(srv.URL)

	resolvers.QueryResolver.User = func(ctx context.Context, id int) (user *User, e error) {
		return &User{}, nil
	}

	t.Run("zero value in nullable field", func(t *testing.T) {
		var resp struct {
			User struct {
				Updated *string
			}
		}

		err := c.Post(`query { user(id: 1) { updated } }`, &resp)
		require.NoError(t, err)

		require.Nil(t, resp.User.Updated)
	})

	t.Run("zero value in non nullable field", func(t *testing.T) {
		var resp struct {
			User struct {
				Created *string
			}
		}

		err := c.Post(`query { user(id: 1) { created } }`, &resp)
		require.EqualError(t, err, `[{"message":"must not be null","path":["user","created"]}]`)
	})

	t.Run("with values", func(t *testing.T) {
		resolvers.QueryResolver.User = func(ctx context.Context, id int) (user *User, e error) {
			updated := time.Date(2010, 1, 1, 0, 0, 20, 0, time.UTC)
			return &User{
				Created: time.Date(2010, 1, 1, 0, 0, 10, 0, time.UTC),
				Updated: &updated,
			}, nil
		}

		var resp struct {
			User struct {
				Created string
				Updated string
			}
		}

		err := c.Post(`query { user(id: 1) { created, updated } }`, &resp)
		require.NoError(t, err)

		require.Equal(t, "2010-01-01T00:00:10Z", resp.User.Created)
		require.Equal(t, "2010-01-01T00:00:20Z", resp.User.Updated)
	})
}
