package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestIntrospection(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		resolvers := &Stub{}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.IntrospectionEnabled(false),
			),
		)

		c := client.New(srv.URL)

		var resp interface{}
		err := c.Post(introspection.Query, &resp)
		require.EqualError(t, err, "[{\"message\":\"introspection disabled\",\"path\":[\"__schema\"]}]")
	})

	t.Run("enabled by default", func(t *testing.T) {
		resolvers := &Stub{}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
			),
		)

		c := client.New(srv.URL)

		var resp interface{}
		err := c.Post(introspection.Query, &resp)
		require.NoError(t, err)

		t.Run("does not return empty deprecation strings", func(t *testing.T) {
			q := `{
			  __type(name:"InnerObject") {
			    fields {
			      name
			      deprecationReason
			    }
			  }
			}`

			c := client.New(srv.URL)
			var resp struct {
				Type struct {
					Fields []struct {
						Name              string
						DeprecationReason *string
					}
				} `json:"__type"`
			}
			err := c.Post(q, &resp)
			require.NoError(t, err)

			require.Equal(t, "id", resp.Type.Fields[0].Name)
			require.Nil(t, resp.Type.Fields[0].DeprecationReason)
		})
	})

	t.Run("disabled by middleware", func(t *testing.T) {
		resolvers := &Stub{}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.RequestMiddleware(func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
					graphql.GetRequestContext(ctx).DisableIntrospection = true

					return next(ctx)
				}),
			),
		)

		c := client.New(srv.URL)

		var resp interface{}
		err := c.Post(introspection.Query, &resp)
		require.EqualError(t, err, "[{\"message\":\"introspection disabled\",\"path\":[\"__schema\"]}]")
	})
}
