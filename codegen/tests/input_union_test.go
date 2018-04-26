package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/codegen"
)

func TestInputUnion(t *testing.T) {
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
