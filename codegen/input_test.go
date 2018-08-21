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
