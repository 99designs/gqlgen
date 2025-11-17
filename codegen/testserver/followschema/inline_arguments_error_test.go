package followschema

import (
	"context"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

// TestInlineArgumentsErrorMessages verifies that validation errors reference
// the inline argument names (what clients wrote), not the bundled parameter name
func TestInlineArgumentsErrorMessages(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.SearchProducts = func(ctx context.Context, filters map[string]any) ([]string, error) {
		return []string{}, nil
	}
	resolvers.QueryResolver.SearchRequired = func(ctx context.Context, filters map[string]any) ([]string, error) {
		return []string{}, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("error references inline arg name when using wrong argument name", func(t *testing.T) {
		var resp struct {
			SearchProducts []string
		}

		// Try to use the bundled parameter name "filters" instead of inline args
		err := c.Post(`query {
			searchProducts(filters: {query: "test"})
		}`, &resp)

		require.Error(t, err)
		t.Logf("Error when using bundled parameter name: %v", err)

		// Error should mention that "filters" is unknown
		// because the schema has been transformed to use query, category, minPrice
		require.Contains(t, err.Error(), "filters",
			"Error should mention 'filters' as unknown argument")
	})

	t.Run("error references inline arg name when using undefined argument", func(t *testing.T) {
		var resp struct {
			SearchProducts []string
		}

		// Try to use an argument that doesn't exist in the input type
		err := c.Post(`query {
			searchProducts(query: "test", unknownArg: "value")
		}`, &resp)

		require.Error(t, err)
		t.Logf("Error when using unknown inline argument: %v", err)

		// Error should mention "unknownArg" (the inline arg that doesn't exist)
		require.Contains(t, err.Error(), "unknownArg",
			"Error should mention 'unknownArg' as the unknown argument")
	})

	t.Run("error references inline arg name when missing required argument", func(t *testing.T) {
		var resp struct {
			SearchRequired []string
		}

		// Try to call searchRequired without the required 'name' field
		err := c.Post(`query {
			searchRequired(age: 30)
		}`, &resp)

		require.Error(t, err)
		t.Logf("Error when missing required inline argument: %v", err)

		// Error should mention "name" (the missing inline argument)
		// NOT "filters" (the bundled parameter that doesn't exist in the query)
		require.Contains(t, err.Error(), "name",
			"Error should mention 'name' as the required argument")
		require.NotContains(t, err.Error(), "filters",
			"Error should NOT mention 'filters' (the bundled parameter name)")
	})

	t.Run("error occurs when wrong type provided for inline arg", func(t *testing.T) {
		var resp struct {
			SearchProducts []string
		}

		// Try to provide wrong type for minPrice (string instead of int)
		err := c.Post(`query {
			searchProducts(minPrice: "not a number")
		}`, &resp)

		require.Error(t, err)
		t.Logf("Error when providing wrong type: %v", err)

		require.Contains(t, err.Error(), "Int cannot represent",
			"Error should be a scalar coercion error for Int type")
	})

	t.Run("error uses inline arg name when required field missing entirely", func(t *testing.T) {
		var resp struct {
			SearchRequired []string
		}

		// Try to call searchRequired with no arguments at all
		err := c.Post(`query {
			searchRequired
		}`, &resp)

		require.Error(t, err)
		t.Logf("Error when all required arguments missing: %v", err)

		// Error should mention the inline argument names (name, age)
		// that are required, NOT the bundled "filters" parameter
		errorMsg := err.Error()
		hasInlineArgName := strings.Contains(errorMsg, "name") || strings.Contains(errorMsg, "age")
		require.True(t, hasInlineArgName,
			"Error should mention required inline argument names (name or age), got: %s", errorMsg)
	})
}
