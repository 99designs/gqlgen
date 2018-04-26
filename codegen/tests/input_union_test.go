package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/codegen"
)

func TestTypeUnionAsInput(t *testing.T) {
	err := codegen.Generate(codegen.Config{
		SchemaStr: `
			type Query {
				addBookmark(b: Bookmarkable!): Boolean!
			}
			type Item {}
			union Bookmarkable = Item
		`,
		ExecFilename:  "gen/inputunion/exec.go",
		ModelFilename: "gen/inputunion/model.go",
	})

	require.EqualError(t, err, "model plan failed: Bookmarkable! cannot be used as argument of Query.addBookmark. only input and scalar types are allowed")
}

func TestTypeInInput(t *testing.T) {
	err := codegen.Generate(codegen.Config{
		SchemaStr: `
			type Query {
				addBookmark(b: BookmarkableInput!): Boolean!
			}
			type Item {}
			input BookmarkableInput {
				item: Item
			}
		`,
		ExecFilename:  "gen/typeinput/exec.go",
		ModelFilename: "gen/typeinput/model.go",
	})

	require.EqualError(t, err, "model plan failed: Item cannot be used as a field of BookmarkableInput. only input and scalar types are allowed")
}
