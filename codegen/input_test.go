package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeUnionAsInput(t *testing.T) {
	err := generate("inputunion", `
		type Query {
			addBookmark(b: Bookmarkable!): Boolean!
		}
		type Item {}
		union Bookmarkable = Item
	`)

	require.EqualError(t, err, "model plan failed: Bookmarkable! cannot be used as argument of Query.addBookmark. only input and scalar types are allowed")
}

func TestTypeInInput(t *testing.T) {
	err := generate("typeinput", `
		type Query {
			addBookmark(b: BookmarkableInput!): Boolean!
		}
		type Item {}
		input BookmarkableInput {
			item: Item
		}
	`)

	require.EqualError(t, err, "model plan failed: Item cannot be used as a field of BookmarkableInput. only input and scalar types are allowed")
}

func TestRawMapInputs(t *testing.T) {
	err := generate("rawmap", `
		type Query {
			mapInput(input: Changes): Boolean
		}
		input Changes {
			a: Int
			b: Int
		}
	`, map[string]string{
		"Changes": "map[string]interface{}",
	})

	require.NoError(t, err)
}

func TestRecursiveInputType(t *testing.T) {
	err := generate("recursiveinput", `
		type Query {
			recursive(input: RecursiveInputSlice): Boolean
		}
		input RecursiveInputSlice {
			self: [RecursiveInputSlice!]
		}
	`, map[string]string{
		"RecursiveInputSlice": "github.com/vektah/gqlgen/codegen/testdata.RecursiveInputSlice",
	})

	require.NoError(t, err)
}

func TestComplexInputTypes(t *testing.T) {
	err := generate("complexinput", `
		type Query {
			nestedInputs(input: [[OuterInput]] = [[{inner: {id: 1}}]]): Boolean
			nestedOutputs: [[OuterObject]]
		}
		input InnerInput {
			id:Int!
		}
		
		input OuterInput {
			inner: InnerInput!
		}
		
		type OuterObject {
			inner: InnerObject!
		}
		
		type InnerObject {
			id: Int!
		}
	`, map[string]string{
		"Changes": "map[string]interface{}",
	})

	require.NoError(t, err)
}
