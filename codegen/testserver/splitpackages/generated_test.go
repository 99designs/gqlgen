//go:generate rm -rf internal/gqlgenexec
//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package splitpackages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestSplitPackagesLayout(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Hello = func(ctx context.Context, name string) (string, error) {
		return "Hello " + name, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{
		Resolvers: resolvers,
	}))
	c := client.New(srv)

	var resp struct {
		Hello string
	}
	c.MustPost(`query { hello(name:"Ada") }`, &resp)
	require.Equal(t, "Hello Ada", resp.Hello)
}

func TestSplitPackagesCodecCompile(t *testing.T) {
	schema := NewExecutableSchema(Config{Resolvers: &Stub{}})
	require.NotNil(t, schema)
}

func TestSplitPackagesCompiles(t *testing.T) {
	schema := NewExecutableSchema(Config{Resolvers: &Stub{}})
	require.NotNil(t, schema)
}

func TestSplitComplexityParity(t *testing.T) {
	t.Run("uses configured complexity handler", func(t *testing.T) {
		schema := NewExecutableSchema(Config{
			Resolvers: &Stub{},
			Complexity: ComplexityRoot{
				Query: struct {
					Hello func(childComplexity int, name string) int
				}{
					Hello: func(childComplexity int, name string) int { return childComplexity + len(name) },
				},
			},
		})

		value, ok := schema.Complexity(context.Background(), "Query", "hello", 4, map[string]any{"name": "Ada"})
		require.True(t, ok)
		require.Equal(t, 7, value)
	})

	t.Run("returns false when complexity function is unset", func(t *testing.T) {
		schema := NewExecutableSchema(Config{Resolvers: &Stub{}})

		value, ok := schema.Complexity(context.Background(), "Query", "hello", 2, map[string]any{"name": "Ada"})
		require.False(t, ok)
		require.Equal(t, 0, value)
	})

	t.Run("returns false on argument parse failure", func(t *testing.T) {
		schema := NewExecutableSchema(Config{
			Resolvers: &Stub{},
			Complexity: ComplexityRoot{
				Query: struct {
					Hello func(childComplexity int, name string) int
				}{
					Hello: func(childComplexity int, name string) int { return childComplexity },
				},
			},
		})

		value, ok := schema.Complexity(context.Background(), "Query", "hello", 3, map[string]any{"name": []int{1}})
		require.False(t, ok)
		require.Equal(t, 0, value)
	})
}
