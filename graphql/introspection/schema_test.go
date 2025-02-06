package introspection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestSchema(t *testing.T) {
	query := &ast.Definition{
		Name: "Query",
		Kind: ast.Object,
	}

	mutation := &ast.Definition{
		Name: "Mutation",
		Kind: ast.Object,
	}

	subscription := &ast.Definition{
		Name: "Subscription",
		Kind: ast.Object,
	}

	directive := &ast.Definition{
		Name: "__Directive",
		Kind: ast.Object,
	}

	schema := &Schema{
		schema: &ast.Schema{
			Query:        query,
			Mutation:     mutation,
			Subscription: subscription,
			Types: map[string]*ast.Definition{
				"Query":       query,
				"Mutation":    mutation,
				"__Directive": directive,
			},
			Description: "test description",
		},
	}

	t.Run("description", func(t *testing.T) {
		require.EqualValues(t, "test description", *schema.Description())
	})

	t.Run("query type", func(t *testing.T) {
		require.Equal(t, "Query", *schema.QueryType().Name())
	})

	t.Run("mutation type", func(t *testing.T) {
		require.Equal(t, "Mutation", *schema.MutationType().Name())
	})

	t.Run("subscription type", func(t *testing.T) {
		require.Equal(t, "Subscription", *schema.SubscriptionType().Name())
	})

	t.Run("types", func(t *testing.T) {
		types := schema.Types()
		require.Len(t, types, 3)
		require.Equal(t, "Mutation", *types[0].Name())
		require.Equal(t, "Query", *types[1].Name())
		require.Equal(t, "__Directive", *types[2].Name())
	})
}
