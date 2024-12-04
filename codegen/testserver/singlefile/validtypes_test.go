package singlefile

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestValidType(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ValidType = func(ctx context.Context) (validType *ValidType, e error) {
		return &ValidType{
			DifferentCase:    "new",
			DifferentCaseOld: "old",
		}, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("fields with differing cases can be distinguished", func(t *testing.T) {
		var resp struct {
			ValidType struct {
				New string `json:"differentCase"`
				Old string `json:"different_case"`
			}
		}
		err := c.Post(`query { validType { differentCase, different_case } }`, &resp)
		require.NoError(t, err)

		require.Equal(t, "new", resp.ValidType.New)
		require.Equal(t, "old", resp.ValidType.Old)
	})
}
