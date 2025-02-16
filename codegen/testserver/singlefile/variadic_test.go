package singlefile

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestVariadic(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.VariadicModel = func(ctx context.Context) (*VariadicModel, error) {
		return &VariadicModel{}, nil
	}
	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	var resp struct {
		VariadicModel struct {
			Value string
		}
	}
	err := c.Post(`query { variadicModel { value(rank: 1) } }`, &resp)
	require.NoError(t, err)
	require.Equal(t, "1", resp.VariadicModel.Value)

	err = c.Post(`query { variadicModel { value(rank: 2) } }`, &resp)
	require.NoError(t, err)
	require.Equal(t, "2", resp.VariadicModel.Value)
}
