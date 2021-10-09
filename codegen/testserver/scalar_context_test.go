package testserver

import (
	"context"
	"math"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

func TestFloatInfAndNaN(t *testing.T) {
	resolvers := &Stub{}

	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers})))

	resolvers.QueryResolver.Rectangle = func(ctx context.Context) (*Rectangle, error) {
		return &Rectangle{
			Length: math.Inf(-1),
			Width:  math.NaN(),
		}, nil
	}

	t.Run("errors on marshaller with context", func(t *testing.T) {
		err := c.Post(`query { rectangle { length width } }`, nil)
		require.Error(t, err)
	})

}
func TestContextPassedToMarshal(t *testing.T) {
	resolvers := &Stub{}

	c := client.New(handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers})))

	resolvers.QueryResolver.Rectangle = func(ctx context.Context) (*Rectangle, error) {
		return &Rectangle{
			Length: math.Inf(-1),
			Width:  math.NaN(),
		}, nil
	}

	t.Run("errors on marshaller with context", func(t *testing.T) {
		err := c.Post(`query { rectangle { length width } }`, nil)
		require.Error(t, err)
	})

}
