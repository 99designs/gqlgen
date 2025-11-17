package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestExpandInlineArguments(t *testing.T) {
	ClearInlineArgsMetadata()

	schemaDoc := `
		directive @inlineArguments on ARGUMENT_DEFINITION
		directive @goModel(model: String) on INPUT_OBJECT

		input SearchArgs @goModel(model: "map[string]any") {
			query: String
			category: String
			minPrice: Float
		}

		type Product {
			id: ID!
			name: String!
		}

		type Query {
			searchProducts(args: SearchArgs @inlineArguments): [Product!]!
		}
	`

	schema, err := gqlparser.LoadSchema(&ast.Source{Input: schemaDoc})
	require.NoError(t, err)

	queryType := schema.Types["Query"]
	require.NotNil(t, queryType)

	// Find searchProducts field (gqlparser adds introspection fields)
	var searchField *ast.FieldDefinition
	for _, f := range queryType.Fields {
		if f.Name == "searchProducts" {
			searchField = f
			break
		}
	}
	require.NotNil(t, searchField)
	require.Len(t, searchField.Arguments, 1)
	require.Equal(t, "args", searchField.Arguments[0].Name)

	err = ExpandInlineArguments(schema)
	require.NoError(t, err)

	queryType = schema.Types["Query"]
	searchField = nil
	for _, f := range queryType.Fields {
		if f.Name == "searchProducts" {
			searchField = f
			break
		}
	}
	require.NotNil(t, searchField)

	// Should now have 3 arguments instead of 1
	require.Len(t, searchField.Arguments, 3, "Arguments should be expanded")
	require.Equal(t, "query", searchField.Arguments[0].Name)
	require.Equal(t, "category", searchField.Arguments[1].Name)
	require.Equal(t, "minPrice", searchField.Arguments[2].Name)

	metadata := GetInlineArgsMetadata("Query", "searchProducts")
	require.NotNil(t, metadata)
	require.Equal(t, "args", metadata.OriginalArgName)
	require.Equal(t, "SearchArgs", metadata.OriginalType)
	require.Equal(t, "map[string]any", metadata.GoType)
	require.Equal(t, []string{"query", "category", "minPrice"}, metadata.ExpandedArgs)
}

func TestExpandInlineArgumentsError(t *testing.T) {
	ClearInlineArgsMetadata()

	schemaDoc := `
		directive @inlineArguments on ARGUMENT_DEFINITION

		type Query {
			test(arg: String @inlineArguments): String
		}
	`

	schema, err := gqlparser.LoadSchema(&ast.Source{Input: schemaDoc})
	require.NoError(t, err)

	err = ExpandInlineArguments(schema)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be an INPUT_OBJECT")
}
