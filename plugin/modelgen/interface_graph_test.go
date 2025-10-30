package modelgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestInterfaceGraph(t *testing.T) {
	// Create a simple schema for testing
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"Node": {
				Name:       "Node",
				Kind:       ast.Interface,
				Interfaces: []string{},
				Fields: []*ast.FieldDefinition{
					{Name: "id"},
				},
			},
			"Element": {
				Name:       "Element",
				Kind:       ast.Interface,
				Interfaces: []string{"Node"},
				Fields: []*ast.FieldDefinition{
					{Name: "id"},
					{Name: "name"},
				},
			},
			"Metal": {
				Name:       "Metal",
				Kind:       ast.Interface,
				Interfaces: []string{"Element"},
				Fields: []*ast.FieldDefinition{
					{Name: "id"},
					{Name: "name"},
					{Name: "atomicNumber"},
				},
			},
		},
	}

	t.Run("builds interface graph correctly", func(t *testing.T) {
		graph := newInterfaceGraph(schema)

		// Check parent relationships
		assert.Empty(t, graph.parentInterfaces["Node"])
		assert.Equal(t, []string{"Node"}, graph.parentInterfaces["Element"])
		assert.Equal(t, []string{"Element"}, graph.parentInterfaces["Metal"])
	})

	t.Run("topological sort orders parents before children", func(t *testing.T) {
		graph := newInterfaceGraph(schema)
		sorted, err := graph.topologicalSort([]string{"Metal", "Node", "Element"})
		require.NoError(t, err)

		// Node should come before Element, Element should come before Metal
		nodeIdx := -1
		elementIdx := -1
		metalIdx := -1
		for i, name := range sorted {
			switch name {
			case "Node":
				nodeIdx = i
			case "Element":
				elementIdx = i
			case "Metal":
				metalIdx = i
			}
		}

		assert.True(t, nodeIdx < elementIdx, "Node should come before Element")
		assert.True(t, elementIdx < metalIdx, "Element should come before Metal")
	})

	t.Run("gets interface own fields correctly", func(t *testing.T) {
		graph := newInterfaceGraph(schema)

		// Node has all its fields as own fields
		nodeFields := graph.getInterfaceOwnFields("Node")
		assert.Len(t, nodeFields, 1)
		assert.Equal(t, "id", nodeFields[0].Name)

		// Element inherits id from Node, only name is own field
		elementFields := graph.getInterfaceOwnFields("Element")
		assert.Len(t, elementFields, 1)
		assert.Equal(t, "name", elementFields[0].Name)

		// Metal inherits id and name, only atomicNumber is own field
		metalFields := graph.getInterfaceOwnFields("Metal")
		assert.Len(t, metalFields, 1)
		assert.Equal(t, "atomicNumber", metalFields[0].Name)
	})
}
