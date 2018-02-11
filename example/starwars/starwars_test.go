package starwars

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/handler"
	introspection "github.com/vektah/gqlgen/neelance/introspection"
)

func TestStarwars(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutor(NewResolver())))
	c := client.New(srv.URL)

	t.Run("Lukes starships", func(t *testing.T) {
		var resp struct {
			Search []struct{ Starships []struct{ Name string } }
		}
		c.MustPost(`{ search(text:"Luke") { ... on Human { starships { name } } } }`, &resp)

		require.Equal(t, "X-Wing", resp.Search[0].Starships[0].Name)
		require.Equal(t, "Imperial shuttle", resp.Search[0].Starships[1].Name)
	})

	t.Run("get character", func(t *testing.T) {
		var resp struct {
			Character struct{ Name string }
		}
		c.MustPost(`{ character(id:2001) { name } }`, &resp)

		require.Equal(t, "R2-D2", resp.Character.Name)
	})

	t.Run("missing character", func(t *testing.T) {
		var resp struct {
			Character *struct{ Name string }
		}
		c.MustPost(`{ character(id:2002) { name } }`, &resp)

		require.Nil(t, resp.Character)
	})

	t.Run("get droid", func(t *testing.T) {
		var resp struct {
			Droid struct{ PrimaryFunction string }
		}
		c.MustPost(`{ droid(id:2001) { primaryFunction } }`, &resp)

		require.Equal(t, "Astromech", resp.Droid.PrimaryFunction)
	})

	t.Run("get human", func(t *testing.T) {
		var resp struct {
			Human struct {
				Starships []struct {
					Name   string
					Length float64
				}
			}
		}
		c.MustPost(`{ human(id:1000) { starships { name length(unit:FOOT) } } }`, &resp)

		require.Equal(t, "X-Wing", resp.Human.Starships[0].Name)
		require.Equal(t, 41.0105, resp.Human.Starships[0].Length)

		require.Equal(t, "Imperial shuttle", resp.Human.Starships[1].Name)
		require.Equal(t, 65.6168, resp.Human.Starships[1].Length)
	})

	t.Run("hero height", func(t *testing.T) {
		var resp struct {
			Hero struct {
				Height float64
			}
		}
		c.MustPost(`{ hero(episode:EMPIRE) { ... on Human { height(unit:METER) } } }`, &resp)

		require.Equal(t, 1.72, resp.Hero.Height)
	})

	t.Run("friends", func(t *testing.T) {
		var resp struct {
			Human struct {
				Friends []struct {
					Name string
				}
			}
		}
		c.MustPost(`{ human(id: 1001) { friends { name } } }`, &resp)

		require.Equal(t, "Wilhuff Tarkin", resp.Human.Friends[0].Name)
	})

	t.Run("friendsConnection.friends", func(t *testing.T) {
		var resp struct {
			Droid struct {
				FriendsConnection struct {
					Friends []struct {
						Name string
					}
				}
			}
		}
		c.MustPost(`{ droid(id:2001) { friendsConnection { friends { name } } } }`, &resp)

		require.Equal(t, "Luke Skywalker", resp.Droid.FriendsConnection.Friends[0].Name)
		require.Equal(t, "Han Solo", resp.Droid.FriendsConnection.Friends[1].Name)
		require.Equal(t, "Leia Organa", resp.Droid.FriendsConnection.Friends[2].Name)
	})

	t.Run("friendsConnection.edges", func(t *testing.T) {
		var resp struct {
			Droid struct {
				FriendsConnection struct {
					Edges []struct {
						Cursor string
						Node   struct {
							Name string
						}
					}
				}
			}
		}
		c.MustPost(`{ droid(id:2001) { friendsConnection { edges { cursor, node { name } } } } }`, &resp)

		require.Equal(t, "Y3Vyc29yMQ==", resp.Droid.FriendsConnection.Edges[0].Cursor)
		require.Equal(t, "Luke Skywalker", resp.Droid.FriendsConnection.Edges[0].Node.Name)
		require.Equal(t, "Y3Vyc29yMg==", resp.Droid.FriendsConnection.Edges[1].Cursor)
		require.Equal(t, "Han Solo", resp.Droid.FriendsConnection.Edges[1].Node.Name)
		require.Equal(t, "Y3Vyc29yMw==", resp.Droid.FriendsConnection.Edges[2].Cursor)
		require.Equal(t, "Leia Organa", resp.Droid.FriendsConnection.Edges[2].Node.Name)
	})

	t.Run("mutations must be run in sequence", func(t *testing.T) {
		var resp struct {
			A struct{ Time string }
			B struct{ Time string }
			C struct{ Time string }
		}

		c.MustPost(`mutation f{
		  a:createReview(episode: NEWHOPE, review:{stars:1, commentary:"Blah blah"})  {
			time
		  }
		  b:createReview(episode: NEWHOPE, review:{stars:1, commentary:"Blah blah"})  {
			time
		  }
		  c:createReview(episode: NEWHOPE, review:{stars:1, commentary:"Blah blah"})  {
			time
		  }
		}`, &resp)

		require.NotEqual(t, resp.A.Time, resp.B.Time)
		require.NotEqual(t, resp.C.Time, resp.B.Time)
	})

	t.Run("multidimensional arrays", func(t *testing.T) {
		var resp struct {
			Starship struct {
				History [][]int
			}
		}
		c.MustPost(`{ starship(id:"3001") { history } }`, &resp)

		require.Len(t, resp.Starship.History, 4)
		require.Len(t, resp.Starship.History[0], 2)
	})

	t.Run("introspection", func(t *testing.T) {
		// Make sure we can run the graphiql introspection query without errors
		c.MustPost(introspection.Query, nil)
	})
}
