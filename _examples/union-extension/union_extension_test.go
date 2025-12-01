package unionextension

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestEvents(t *testing.T) {
	srv := handler.New(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	query := `{
		events {
			... on Like {
				from
			}
			... on Post {
				message
			}
		}
	}
	`

	var resp struct {
		Events []struct {
			From    string
			Message string
		}
	}
	c.MustPost(query, &resp)

	require.Equal(t, "John", resp.Events[0].From)
	require.Equal(t, "Hello", resp.Events[1].Message)
}

func TestCachedEvents(t *testing.T) {
	srv := handler.New(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	query := `{
		cachedEvents {
			... on Like {
				from
			}
			... on Post {
				message
			}
		}
	}
	`

	var resp struct {
		CachedEvents []struct {
			From    string
			Message string
		}
	}
	c.MustPost(query, &resp)

	require.Equal(t, "CachedLike", resp.CachedEvents[0].From)
	require.Equal(t, "CachedPost", resp.CachedEvents[1].Message)
}
