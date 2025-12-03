package singlefile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/introspection"
)

func TestIntrospection(t *testing.T) {
	t.Run("disabled when creating your own server", func(t *testing.T) {
		resolvers := &Stub{}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
		srv.AddTransport(transport.POST{})
		c := client.New(srv)

		var resp any
		err := c.Post(introspection.Query, &resp)
		require.EqualError(
			t,
			err,
			"[{\"message\":\"introspection disabled\",\"path\":[\"__schema\"]}]",
		)
	})

	t.Run("enabled by default", func(t *testing.T) {
		resolvers := &Stub{}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
		srv.AddTransport(transport.POST{})
		srv.Use(extension.Introspection{})

		c := client.New(srv)

		var resp any
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

		t.Run("deprecated arguments", func(t *testing.T) {
			var resp struct {
				Type struct {
					Fields []struct {
						Name string
						Args []struct {
							Name              string
							DeprecationReason *string
						}
					}
				} `json:"__type"`
			}

			err := c.Post(
				`{ __type(name:"Query") { fields { name args { name deprecationReason }}}}`,
				&resp,
			)
			require.NoError(t, err)

			var args []struct {
				Name              string
				DeprecationReason *string
			}
			for _, f := range resp.Type.Fields {
				if f.Name == "fieldWithDeprecatedArg" {
					args = f.Args
					break
				}
			}

			require.Len(t, args, 2)
			require.Equal(t, "oldArg", args[0].Name)
			require.NotNil(t, args[0].DeprecationReason)
			require.Equal(t, "old arg", *args[0].DeprecationReason)

			require.Equal(t, "newArg", args[1].Name)
			require.Nil(t, args[1].DeprecationReason)
		})
	})

	t.Run("disabled by middleware", func(t *testing.T) {
		resolvers := &Stub{}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
		srv.AddTransport(transport.POST{})
		srv.AroundOperations(
			func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
				graphql.GetOperationContext(ctx).DisableIntrospection = true
				return next(ctx)
			},
		)
		c := client.New(srv)

		var resp any
		err := c.Post(introspection.Query, &resp)
		require.EqualError(
			t,
			err,
			"[{\"message\":\"introspection disabled\",\"path\":[\"__schema\"]}]",
		)
	})
}
