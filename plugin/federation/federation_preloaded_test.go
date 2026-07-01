//go:generate go run ../../testdata/gqlgen.go -config testdata/entityresolverpreloaded/gqlgen.yml
package federation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/plugin/federation/testdata/entityresolverpreloaded"
	preloadedgen "github.com/99designs/gqlgen/plugin/federation/testdata/entityresolverpreloaded/generated"
)

// TestEntityResolverPreloaded is the runtime proof of N+1 (b): with
// preloaded_requires, the batch resolver receives every
// product's @requires data (Category) in a single scope. The resolver builds
// each product's `display` from the whole batch of categories, so the assertion
// — that every product's display lists all categories — can only pass if all
// representations' @requires data was visible at once inside one resolver call.
func TestEntityResolverPreloaded(t *testing.T) {
	srv := handler.New(
		preloadedgen.NewExecutableSchema(preloadedgen.Config{
			Resolvers: &entityresolverpreloaded.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	representations := []map[string]any{
		{"__typename": "Product", "id": "1", "category": "books"},
		{"__typename": "Product", "id": "2", "category": "electronics"},
		{"__typename": "Product", "id": "3", "category": "toys"},
	}

	var resp struct {
		Entities []struct {
			ID      string `json:"id"`
			Display string `json:"display"`
		} `json:"_entities"`
	}

	err := c.Post(
		entityQuery([]string{"Product { id display }"}),
		&resp,
		client.Var("representations", representations),
	)
	require.NoError(t, err)
	require.Len(t, resp.Entities, 3)

	// Order is preserved, and each product's display reflects the entire batch
	// of categories — i.e. the resolver saw every @requires value at once.
	const wantBatch = "batch of 3: books,electronics,toys"
	require.Equal(t, "1", resp.Entities[0].ID)
	require.Equal(t, "books display ("+wantBatch+")", resp.Entities[0].Display)
	require.Equal(t, "2", resp.Entities[1].ID)
	require.Equal(t, "electronics display ("+wantBatch+")", resp.Entities[1].Display)
	require.Equal(t, "3", resp.Entities[2].ID)
	require.Equal(t, "toys display ("+wantBatch+")", resp.Entities[2].Display)
}

// TestEntityResolverPreloadedPerIndexError proves PR 3: a single
// entity in the batch can fail without sinking its siblings. The resolver
// returns a graphql.BatchErrorList; the runtime nulls just the failed entity,
// reports its error against _entities[index], and resolves the rest.
func TestEntityResolverPreloadedPerIndexError(t *testing.T) {
	srv := handler.New(
		preloadedgen.NewExecutableSchema(preloadedgen.Config{
			Resolvers: &entityresolverpreloaded.Resolver{},
		}),
	)
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	representations := []map[string]any{
		{"__typename": "Product", "id": "1", "category": "books"},
		{"__typename": "Product", "id": "2", "category": "error"},
		{"__typename": "Product", "id": "3", "category": "toys"},
	}

	var resp struct {
		Entities []struct {
			ID      string `json:"id"`
			Display string `json:"display"`
		} `json:"_entities"`
	}

	err := c.Post(
		entityQuery([]string{"Product { id display }"}),
		&resp,
		client.Var("representations", representations),
	)

	// One entity failed, so the response carries a GraphQL error...
	require.Error(t, err)
	var gqlErrs []struct {
		Message string `json:"message"`
		Path    []any  `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(err.Error()), &gqlErrs))
	require.Len(t, gqlErrs, 1)
	assert.Equal(t, "cannot display product 2", gqlErrs[0].Message)
	// ...attributed to the failed entity's position, not the whole field.
	assert.Equal(t, []any{"_entities", float64(1)}, gqlErrs[0].Path)

	// ...but its siblings still resolved.
	require.Len(t, resp.Entities, 3)
	assert.Equal(t, "1", resp.Entities[0].ID)
	assert.Equal(t, "books display (batch of 3: books,error,toys)", resp.Entities[0].Display)
	assert.Empty(t, resp.Entities[1].ID) // the failed entity is null
	assert.Equal(t, "3", resp.Entities[2].ID)
	assert.Equal(t, "toys display (batch of 3: books,error,toys)", resp.Entities[2].Display)
}
