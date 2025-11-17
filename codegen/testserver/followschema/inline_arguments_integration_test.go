package followschema

import (
	"context"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"
)

func TestInlineArguments(t *testing.T) {
	resolvers := &Stub{}

	// Implement the SearchProducts resolver using the Stub
	resolvers.QueryResolver.SearchProducts = func(ctx context.Context, filters map[string]any) ([]string, error) {
		results := []string{}
		if query, ok := filters["query"].(*string); ok && query != nil {
			results = append(results, "query:"+*query)
		}
		if category, ok := filters["category"].(*string); ok && category != nil {
			results = append(results, "category:"+*category)
		}
		if minPrice, ok := filters["minPrice"].(*int); ok && minPrice != nil {
			results = append(results, fmt.Sprintf("minPrice:%d", *minPrice))
		}
		return results, nil
	}

	// Implement the SearchProductsNormal resolver using the Stub
	resolvers.QueryResolver.SearchProductsNormal = func(ctx context.Context, filters map[string]any) ([]string, error) {
		results := []string{}
		if query, ok := filters["query"].(*string); ok && query != nil {
			results = append(results, "query:"+*query)
		}
		if category, ok := filters["category"].(*string); ok && category != nil {
			results = append(results, "category:"+*category)
		}
		if minPrice, ok := filters["minPrice"].(*int); ok && minPrice != nil {
			results = append(results, fmt.Sprintf("minPrice:%d", *minPrice))
		}
		return results, nil
	}

	// Implement the SearchRequired resolver for testing required arguments
	resolvers.QueryResolver.SearchRequired = func(ctx context.Context, filters map[string]any) ([]string, error) {
		results := []string{}
		if name, ok := filters["name"].(string); ok {
			results = append(results, "name:"+name)
		}
		if age, ok := filters["age"].(int); ok {
			results = append(results, fmt.Sprintf("age:%d", age))
		}
		return results, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})
	c := client.New(srv)

	t.Run("with @inlineArguments directive - flat query syntax", func(t *testing.T) {
		var resp struct {
			SearchProducts []string
		}
		// Note: With @inlineArguments, we can use flat argument syntax
		c.MustPost(`query {
			searchProducts(query: "laptop", category: "electronics", minPrice: 500)
		}`, &resp)

		require.Len(t, resp.SearchProducts, 3)
		require.Contains(t, resp.SearchProducts, "query:laptop")
		require.Contains(t, resp.SearchProducts, "category:electronics")
		require.Contains(t, resp.SearchProducts, "minPrice:500")
	})

	t.Run("without @inlineArguments directive - wrapped query syntax", func(t *testing.T) {
		var resp struct {
			SearchProductsNormal []string
		}
		// Note: Without @inlineArguments, we must use nested input object syntax
		c.MustPost(`query {
			searchProductsNormal(filters: {query: "laptop", category: "electronics", minPrice: 500})
		}`, &resp)

		require.Len(t, resp.SearchProductsNormal, 3)
		require.Contains(t, resp.SearchProductsNormal, "query:laptop")
		require.Contains(t, resp.SearchProductsNormal, "category:electronics")
		require.Contains(t, resp.SearchProductsNormal, "minPrice:500")
	})

	t.Run("with @inlineArguments - partial arguments", func(t *testing.T) {
		var resp struct {
			SearchProducts []string
		}
		// Only provide some of the optional arguments
		c.MustPost(`query {
			searchProducts(query: "phone")
		}`, &resp)

		require.Len(t, resp.SearchProducts, 1)
		require.Contains(t, resp.SearchProducts, "query:phone")
	})

	t.Run("with @inlineArguments - no arguments", func(t *testing.T) {
		var resp struct {
			SearchProducts []string
		}
		// Call with no arguments at all
		c.MustPost(`query {
			searchProducts
		}`, &resp)

		require.Empty(t, resp.SearchProducts)
	})

	t.Run("with @inlineArguments - required input type with all required fields", func(t *testing.T) {
		var resp struct {
			SearchRequired []string
		}
		// Required fields must be provided as individual arguments
		c.MustPost(`query {
			searchRequired(name: "John", age: 30)
		}`, &resp)

		require.Len(t, resp.SearchRequired, 2)
		require.Contains(t, resp.SearchRequired, "name:John")
		require.Contains(t, resp.SearchRequired, "age:30")
	})

	t.Run("mutation with @inlineArguments", func(t *testing.T) {
		// Implement the UpdateProduct mutation resolver
		resolvers.MutationResolver.UpdateProduct = func(ctx context.Context, input map[string]any) (string, error) {
			result := "Updated product"
			if id, ok := input["id"].(string); ok {
				result += " ID:" + id
			}
			if name, ok := input["name"].(*string); ok && name != nil {
				result += " Name:" + *name
			}
			if price, ok := input["price"].(*float64); ok && price != nil {
				result += fmt.Sprintf(" Price:%.2f", *price)
			}
			return result, nil
		}

		var resp struct {
			UpdateProduct string
		}

		// Mutation with flat argument syntax
		c.MustPost(`mutation {
			updateProduct(id: "123", name: "New Product", price: 99.99)
		}`, &resp)

		require.Contains(t, resp.UpdateProduct, "ID:123")
		require.Contains(t, resp.UpdateProduct, "Name:New Product")
		require.Contains(t, resp.UpdateProduct, "Price:99.99")
	})

	t.Run("default values on inlined arguments", func(t *testing.T) {
		// Implement the SearchWithDefaults resolver
		resolvers.QueryResolver.SearchWithDefaults = func(ctx context.Context, filters map[string]any) ([]string, error) {
			results := []string{}
			if query, ok := filters["query"].(*string); ok && query != nil {
				results = append(results, "query:"+*query)
			}
			if limit, ok := filters["limit"].(*int); ok && limit != nil {
				results = append(results, fmt.Sprintf("limit:%d", *limit))
			}
			if includeArchived, ok := filters["includeArchived"].(*bool); ok && includeArchived != nil {
				results = append(results, fmt.Sprintf("includeArchived:%v", *includeArchived))
			}
			return results, nil
		}

		var resp struct {
			SearchWithDefaults []string
		}

		// Test 1: Call with no arguments - should receive defaults
		c.MustPost(`query {
			searchWithDefaults
		}`, &resp)

		require.Len(t, resp.SearchWithDefaults, 3)
		require.Contains(t, resp.SearchWithDefaults, "query:default search")
		require.Contains(t, resp.SearchWithDefaults, "limit:20")
		require.Contains(t, resp.SearchWithDefaults, "includeArchived:false")

		// Test 2: Partial arguments - should merge provided + defaults
		c.MustPost(`query {
			searchWithDefaults(query: "laptop")
		}`, &resp)

		require.Len(t, resp.SearchWithDefaults, 3)
		require.Contains(t, resp.SearchWithDefaults, "query:laptop")
		require.Contains(t, resp.SearchWithDefaults, "limit:20")              // default
		require.Contains(t, resp.SearchWithDefaults, "includeArchived:false") // default

		// Test 3: Override all defaults
		c.MustPost(`query {
			searchWithDefaults(query: "phone", limit: 50, includeArchived: true)
		}`, &resp)

		require.Len(t, resp.SearchWithDefaults, 3)
		require.Contains(t, resp.SearchWithDefaults, "query:phone")
		require.Contains(t, resp.SearchWithDefaults, "limit:50")
		require.Contains(t, resp.SearchWithDefaults, "includeArchived:true")
	})

	t.Run("mixed inline and regular arguments", func(t *testing.T) {
		// Implement the SearchMixed resolver
		resolvers.QueryResolver.SearchMixed = func(ctx context.Context, filters map[string]any, limit *int, offset *int, sortBy *string) ([]string, error) {
			results := []string{}

			// Process inlined filters
			if query, ok := filters["query"].(*string); ok && query != nil {
				results = append(results, "query:"+*query)
			}
			if category, ok := filters["category"].(*string); ok && category != nil {
				results = append(results, "category:"+*category)
			}

			// Process regular arguments
			if limit != nil {
				results = append(results, fmt.Sprintf("limit:%d", *limit))
			}
			if offset != nil {
				results = append(results, fmt.Sprintf("offset:%d", *offset))
			}
			if sortBy != nil {
				results = append(results, "sortBy:"+*sortBy)
			}

			return results, nil
		}

		var resp struct {
			SearchMixed []string
		}

		// Test 1: All arguments (inlined + regular)
		c.MustPost(`query {
			searchMixed(query: "laptop", category: "electronics", limit: 5, offset: 10, sortBy: "price")
		}`, &resp)

		require.Len(t, resp.SearchMixed, 5)
		require.Contains(t, resp.SearchMixed, "query:laptop")
		require.Contains(t, resp.SearchMixed, "category:electronics")
		require.Contains(t, resp.SearchMixed, "limit:5")
		require.Contains(t, resp.SearchMixed, "offset:10")
		require.Contains(t, resp.SearchMixed, "sortBy:price")

		// Test 2: Only inlined arguments
		c.MustPost(`query {
			searchMixed(query: "phone")
		}`, &resp)

		require.Len(t, resp.SearchMixed, 3) // query + default limit + default offset
		require.Contains(t, resp.SearchMixed, "query:phone")
		require.Contains(t, resp.SearchMixed, "limit:10") // default
		require.Contains(t, resp.SearchMixed, "offset:0") // default

		// Test 3: Only regular arguments (no inlined)
		c.MustPost(`query {
			searchMixed(limit: 100, sortBy: "name")
		}`, &resp)

		require.Len(t, resp.SearchMixed, 3) // limit + offset default + sortBy
		require.Contains(t, resp.SearchMixed, "limit:100")
		require.Contains(t, resp.SearchMixed, "offset:0") // default
		require.Contains(t, resp.SearchMixed, "sortBy:name")
	})

	t.Run("schema reuse - same input type on multiple fields", func(t *testing.T) {
		resolvers.QueryResolver.SearchProducts = func(ctx context.Context, filters map[string]any) ([]string, error) {
			if query, ok := filters["query"].(*string); ok && query != nil {
				return []string{"searchProducts:" + *query}, nil
			}
			return []string{}, nil
		}

		resolvers.QueryResolver.FilterProducts = func(ctx context.Context, filters map[string]any) ([]string, error) {
			if query, ok := filters["query"].(*string); ok && query != nil {
				return []string{"filterProducts:" + *query}, nil
			}
			return []string{}, nil
		}

		resolvers.QueryResolver.FindProducts = func(ctx context.Context, filters map[string]any) ([]string, error) {
			if query, ok := filters["query"].(*string); ok && query != nil {
				return []string{"findProducts:" + *query}, nil
			}
			return []string{}, nil
		}

		// Test all three fields in a single query
		var resp struct {
			SearchProducts []string
			FilterProducts []string
			FindProducts   []string
		}

		c.MustPost(`query {
			searchProducts(query: "laptop")
			filterProducts(query: "phone")
			findProducts(query: "tablet")
		}`, &resp)

		require.Len(t, resp.SearchProducts, 1)
		require.Contains(t, resp.SearchProducts[0], "laptop")

		require.Len(t, resp.FilterProducts, 1)
		require.Contains(t, resp.FilterProducts[0], "phone")

		require.Len(t, resp.FindProducts, 1)
		require.Contains(t, resp.FindProducts[0], "tablet")
	})

	t.Run("introspection shows flat arguments (not bundled input)", func(t *testing.T) {
		var resp struct {
			Schema struct {
				QueryType struct {
					Fields []struct {
						Name string
						Args []struct {
							Name string
							Type struct {
								Kind   string
								Name   *string
								OfType *struct {
									Kind string
									Name *string
								}
							}
						}
					}
				}
			} `json:"__schema"`
		}

		// Query introspection to see what arguments are exposed
		c.MustPost(`{
			__schema {
				queryType {
					fields {
						name
						args {
							name
							type {
								kind
								name
								ofType {
									kind
									name
								}
							}
						}
					}
				}
			}
		}`, &resp)

		// Find the searchProducts field
		var searchField *struct {
			Name string
			Args []struct {
				Name string
				Type struct {
					Kind   string
					Name   *string
					OfType *struct {
						Kind string
						Name *string
					}
				}
			}
		}
		for i := range resp.Schema.QueryType.Fields {
			if resp.Schema.QueryType.Fields[i].Name == "searchProducts" {
				searchField = &resp.Schema.QueryType.Fields[i]
				break
			}
		}
		require.NotNil(t, searchField, "searchProducts field should exist in introspection")

		// Verify we see the FLAT arguments (query, category, minPrice)
		// NOT a single bundled 'filters' argument
		expectedArgs := map[string]bool{
			"query":    true,
			"category": true,
			"minPrice": true,
		}

		actualArgs := make(map[string]bool)
		for _, arg := range searchField.Args {
			actualArgs[arg.Name] = true
		}

		require.Equal(t, expectedArgs, actualArgs,
			"Introspection should show individual arguments (query, category, minPrice), not bundled 'filters' input")

		// Verify we DON'T see the original 'filters' argument
		for _, arg := range searchField.Args {
			require.NotEqual(t, "filters", arg.Name,
				"Should not see the original bundled 'filters' argument in introspection")
		}
	})

	t.Run("fields with deprecated directive can be inlined", func(t *testing.T) {
		resolvers.QueryResolver.SearchWithDirectives = func(ctx context.Context, input map[string]any) ([]string, error) {
			results := []string{}
			if oldField, ok := input["oldField"].(*string); ok && oldField != nil {
				results = append(results, "oldField:"+*oldField)
			}
			if newField, ok := input["newField"].(*string); ok && newField != nil {
				results = append(results, "newField:"+*newField)
			}
			return results, nil
		}

		var queryResp struct {
			SearchWithDirectives []string
		}
		c.MustPost(`query {
			searchWithDirectives(oldField: "old", newField: "new")
		}`, &queryResp)

		require.Len(t, queryResp.SearchWithDirectives, 2)
		require.Contains(t, queryResp.SearchWithDirectives, "oldField:old")
		require.Contains(t, queryResp.SearchWithDirectives, "newField:new")
	})
}
