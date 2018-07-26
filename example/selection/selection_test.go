package selection

import (
	"net/http/httptest"
	"testing"

	"github.com/vektah/gqlgen/client"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/handler"
)

func TestSelection(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: &Resolver{}})))
	c := client.New(srv.URL)

	query := `{
			events {
				selection
				collected

				... on Post {
					message
					sent
				}

				...LikeFragment
			}
		}
		fragment LikeFragment on Like { reaction sent }
		`

	var resp struct {
		Events []struct {
			Selection []string
			Collected []string

			Message  string
			Reaction string
			Sent     string
		}
	}
	c.MustPost(query, &resp)

	require.Equal(t, []string{
		"selection as selection",
		"collected as collected",
		"inline fragment on Post",
		"named fragment LikeFragment on Like",
	}, resp.Events[0].Selection)

	require.Equal(t, []string{
		"selection as selection",
		"collected as collected",
		"reaction as reaction",
		"sent as sent",
	}, resp.Events[0].Collected)
}
