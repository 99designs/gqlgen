package testserver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestEmbedded(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.EmbeddedCase1 = func(ctx context.Context) (*EmbeddedCase1, error) {
		return &EmbeddedCase1{}, nil
	}
	resolver.QueryResolver.EmbeddedCase2 = func(ctx context.Context) (*EmbeddedCase2, error) {
		return &EmbeddedCase2{&unexportedEmbeddedPointer{}}, nil
	}

	c := client.New(handler.GraphQL(
		NewExecutableSchema(Config{Resolvers: resolver}),
	))

	t.Run("embedded case 1", func(t *testing.T) {
		var resp struct {
			EmbeddedCase1 struct {
				ExportedEmbeddedPointerExportedMethod string
			}
		}
		err := c.Post(`query { embeddedCase1 { exportedEmbeddedPointerExportedMethod } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, resp.EmbeddedCase1.ExportedEmbeddedPointerExportedMethod, "ExportedEmbeddedPointerExportedMethodResponse")
	})

	t.Run("embedded case 2", func(t *testing.T) {
		var resp struct {
			EmbeddedCase2 struct {
				UnexportedEmbeddedPointerExportedMethod string
			}
		}
		err := c.Post(`query { embeddedCase2 { unexportedEmbeddedPointerExportedMethod } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, resp.EmbeddedCase2.UnexportedEmbeddedPointerExportedMethod, "UnexportedEmbeddedPointerExportedMethodResponse")
	})
}
