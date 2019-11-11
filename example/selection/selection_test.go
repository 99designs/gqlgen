package selection

import (
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

func TestSelection(t *testing.T) {
	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}})))

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
