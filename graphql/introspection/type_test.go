package introspection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/ast"
)

func TestType(t *testing.T) {
	schemaType := Type{
		def: &ast.Definition{
			Name:        "Query",
			Description: "test description",
			Fields: ast.FieldList{
				&ast.FieldDefinition{Name: "__schema"},
				&ast.FieldDefinition{Name: "test"},
			},
			Kind: ast.Object,
		},
	}

	t.Run("name", func(t *testing.T) {
		require.Equal(t, "Query", *schemaType.Name())
	})

	t.Run("description", func(t *testing.T) {
		require.Equal(t, "test description", schemaType.Description())
	})

	t.Run("fields ", func(t *testing.T) {
		fields := schemaType.Fields(false)
		require.Len(t, fields, 1)
		require.Equal(t, "test", fields[0].Name)
	})
}
