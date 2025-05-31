package graphql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestFieldSet_MarshalGQL(t *testing.T) {
	t.Run("Should_Deduplicate_Keys", func(t *testing.T) {
		fs := NewFieldSet([]CollectedField{
			{Field: &ast.Field{Alias: "__typename"}},
			{Field: &ast.Field{Alias: "__typename"}},
		})
		fs.Values[0] = MarshalString("A")
		fs.Values[1] = MarshalString("A")

		b := bytes.NewBuffer(nil)
		fs.MarshalGQL(b)

		assert.Equal(t, "{\"__typename\":\"A\"}", string(b.Bytes()))
	})
}
