package scalars

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/handler"
	introspection "github.com/vektah/gqlgen/neelance/introspection"
)

type RawUser struct {
	ID       string
	Name     string
	Created  int64
	Location string
}

func TestScalars(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(&Resolver{})))
	c := client.New(srv.URL)

	t.Run("marshaling", func(t *testing.T) {
		var resp struct {
			User   RawUser
			Search []RawUser
		}
		c.MustPost(`{
				user(id:"1") {
					...UserData
				}
				search(input:{location:"6,66", createdAfter:666}) {
					...UserData
				}
			}
			fragment UserData on User  { id name created location }`, &resp)

		require.Equal(t, "1,2", resp.User.Location)
		require.Equal(t, time.Now().Unix(), resp.User.Created)
		require.Equal(t, "6,66", resp.Search[0].Location)
		require.Equal(t, int64(666), resp.Search[0].Created)
	})

	t.Run("default search location", func(t *testing.T) {
		var resp struct{ Search []RawUser }

		err := c.Post(`{ search { location } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "37,144", resp.Search[0].Location)
	})

	t.Run("test custom error messages", func(t *testing.T) {
		var resp struct{ Search []RawUser }

		err := c.Post(`{ search(input:{createdAfter:"2014"}) { id } }`, &resp)
		require.EqualError(t, err, "errors: [graphql: time should be a unix timestamp]")
	})

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		var resp interface{}
		c.MustPost(introspection.Query, &resp)
	})
}
