package scalars

import (
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/stretchr/testify/require"
)

type RawUser struct {
	ID                string
	Name              string
	Created           int64
	Address           struct{ Location string }
	PrimitiveResolver string
	CustomResolver    string
	Tier              string
	CarManufacturer   string
	IsBanned          bool
	IsLoginBanned     bool
	IsQueryBanned     bool
	Children          int
	Cars              int
	Weddings          int
	SomeBytes         string
	SomeOtherBytes    string
	SomeRunes         string
	RemoteBytes       string
	RemoteRunes       string
}

func TestScalars(t *testing.T) {
	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}})))

	t.Run("marshaling", func(t *testing.T) {
		var resp struct {
			User   RawUser
			Search []RawUser
		}
		c.MustPost(`{
				user(id:"=1=") {
					...UserData
				}
				search(input:{location:"6,66", createdAfter:666}) {
					...UserData
				}
			}
			fragment UserData on User  { id name created tier address { location } }`, &resp)

		require.Equal(t, "1,2", resp.User.Address.Location)
		// There can be a delay between creation and test assertion, so we
		// give some leeway to eliminate false positives.
		require.WithinDuration(t, time.Now(), time.Unix(resp.User.Created, 0), 5*time.Second)
		require.Equal(t, "6,66", resp.Search[0].Address.Location)
		require.Equal(t, int64(666), resp.Search[0].Created)
		require.Equal(t, "A", resp.Search[0].Tier)
	})

	t.Run("default search location", func(t *testing.T) {
		var resp struct{ Search []RawUser }

		err := c.Post(`{ search {  address { location }  } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "37,144", resp.Search[0].Address.Location)
	})

	t.Run("remote named string", func(t *testing.T) {
		var resp struct{ User RawUser }

		err := c.Post(`{ user(id:"=1=") { carManufacturer } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "TESLA", resp.User.CarManufacturer)
	})

	t.Run("alias declaration and type definition", func(t *testing.T) {
		var resp struct{ User RawUser }

		err := c.Post(`{ user(id:"=1=") { isBanned isLoginBanned isQueryBanned } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, false, resp.User.IsBanned)
		require.Equal(t, true, resp.User.IsLoginBanned)
		require.Equal(t, true, resp.User.IsQueryBanned)
	})

	t.Run("unusual basic", func(t *testing.T) {
		var resp struct{ User RawUser }

		err := c.Post(`{ user(id:"=1=") { children cars weddings } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, 3, resp.User.Children)
		require.Equal(t, 5, resp.User.Cars)
		require.Equal(t, 2, resp.User.Weddings)
	})

	t.Run("basic aliases byte and rune", func(t *testing.T) {
		var resp struct{ User RawUser }

		err := c.Post(`{ user(id:"=1=") { someBytes someOtherBytes someRunes remoteBytes remoteRunes } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "abcdef", resp.User.SomeBytes)
		require.Equal(t, "abcdef", resp.User.SomeOtherBytes)
		require.Equal(t, "Hello 世界", resp.User.SomeRunes)
		require.Equal(t, "fedcba", resp.User.RemoteBytes)
		require.Equal(t, "界世 Hello", resp.User.RemoteRunes)
	})

	t.Run("custom error messages", func(t *testing.T) {
		var resp struct{ Search []RawUser }

		err := c.Post(`{ search(input:{createdAfter:"2014"}) { id } }`, &resp)
		require.EqualError(t, err, `[{"message":"time should be a unix timestamp","path":["search","input","createdAfter"]}]`)
	})

	t.Run("scalar resolver methods", func(t *testing.T) {
		var resp struct{ User RawUser }
		c.MustPost(`{ user(id: "=1=") { primitiveResolver, customResolver } }`, &resp)

		require.Equal(t, "test", resp.User.PrimitiveResolver)
		require.Equal(t, "5,1", resp.User.CustomResolver)
	})

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		var resp interface{}
		c.MustPost(introspection.Query, &resp)
	})
}
