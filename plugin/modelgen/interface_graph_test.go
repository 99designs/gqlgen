package modelgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestInterfaceGraph(t *testing.T) {
	schema := createNodeElementMetalSchema("Node", "Element", "Metal")

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

		assert.Less(t, nodeIdx, elementIdx, "Node should come before Element")
		assert.Less(t, elementIdx, metalIdx, "Element should come before Metal")
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

	t.Run("includes all interfaces in graph", func(t *testing.T) {
		schemaWithMixed := createNodeElementMetalSchema("Node", "Metal")
		graph := newInterfaceGraph(schemaWithMixed)

		// All interfaces should be in graph (graph stores all interfaces)
		_, nodeExists := graph.parentInterfaces["Node"]
		assert.True(t, nodeExists, "Node should be in graph")

		_, elementExists := graph.parentInterfaces["Element"]
		assert.True(t, elementExists, "Element should be in graph (even without directive)")

		_, metalExists := graph.parentInterfaces["Metal"]
		assert.True(t, metalExists, "Metal should be in graph")

		// But isEmbeddable should filter by directive
		assert.True(t, graph.isEmbeddable("Node"), "Node should be embeddable (has directive)")
		assert.False(t, graph.isEmbeddable("Element"), "Element should NOT be embeddable (no directive)")
		assert.True(t, graph.isEmbeddable("Metal"), "Metal should be embeddable (has directive)")
	})

	t.Run("isEmbeddable returns true for interfaces with directive", func(t *testing.T) {
		schema := createNodeElementMetalSchema("Node")
		graph := newInterfaceGraph(schema)

		assert.True(t, graph.isEmbeddable("Node"), "Node should be embeddable")
	})

	t.Run("isEmbeddable returns false for interfaces without directive", func(t *testing.T) {
		schema := createNodeElementMetalSchema("Node")
		graph := newInterfaceGraph(schema)

		assert.False(t, graph.isEmbeddable("Element"), "Element should not be embeddable")
	})

	t.Run("isEmbeddable returns false for non-existent interfaces", func(t *testing.T) {
		schema := createNodeElementMetalSchema("Node")
		graph := newInterfaceGraph(schema)

		assert.False(t, graph.isEmbeddable("NonExistent"), "Non-existent interface should not be embeddable")
	})
}

func TestInterfaceGraphGetEmbeddableParents(t *testing.T) {
	testCases := []struct {
		name                    string
		schema                  *ast.Schema
		interfaceName           string
		expectedParents         []string
		unexpectedParents       []string
		expectedSkippedFields   []string
		shouldHaveSkippedFields bool
	}{
		{
			name:                    "all parents have directive",
			schema:                  createABCChainSchema("A", "B", "C"),
			interfaceName:           "C",
			expectedParents:         []string{"B"},
			unexpectedParents:       []string{"A"},
			shouldHaveSkippedFields: false,
		},
		{
			name:                    "some parents missing directive",
			schema:                  createABCChainSchema("A", "C"),
			interfaceName:           "C",
			expectedParents:         []string{"A"},
			unexpectedParents:       []string{"B"},
			expectedSkippedFields:   []string{"fieldB"},
			shouldHaveSkippedFields: true,
		},
		{
			name:                    "deep inheritance chain with mixed directives",
			schema:                  createABCDChainSchema("A", "D"),
			interfaceName:           "D",
			expectedParents:         []string{"A"},
			unexpectedParents:       []string{"B", "C"},
			expectedSkippedFields:   []string{"fieldB", "fieldC"},
			shouldHaveSkippedFields: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			graph := newInterfaceGraph(tc.schema)
			info := graph.getEmbeddingInfo(tc.interfaceName)

			for _, expected := range tc.expectedParents {
				assert.Contains(t, info.Parents, expected, "should contain parent %s", expected)
			}

			for _, unexpected := range tc.unexpectedParents {
				assert.NotContains(t, info.Parents, unexpected, "should not contain parent %s", unexpected)
			}

			if tc.shouldHaveSkippedFields {
				assert.NotEmpty(t, info.SkippedFields, "should have skipped fields")
				fieldNames := make(map[string]bool)
				for _, field := range info.SkippedFields {
					fieldNames[field.Name] = true
				}
				for _, expectedField := range tc.expectedSkippedFields {
					assert.True(t, fieldNames[expectedField], "should contain skipped field %s", expectedField)
				}
			} else {
				assert.Empty(t, info.SkippedFields, "should not have skipped fields")
			}
		})
	}
}

// createNodeElementMetalSchema creates a Node->Element->Metal hierarchy.
// embeddable specifies which interfaces should have the goEmbedInterface directive.
func createNodeElementMetalSchema(embeddable ...string) *ast.Schema {
	embeddableSet := make(map[string]bool)
	for _, name := range embeddable {
		embeddableSet[name] = true
	}

	hasDirective := func(name string) bool {
		return embeddableSet[name]
	}

	return &ast.Schema{
		Types: map[string]*ast.Definition{
			"Node": {
				Name:       "Node",
				Kind:       ast.Interface,
				Interfaces: []string{},
				Fields:     []*ast.FieldDefinition{{Name: "id"}},
				Directives: directives(embeddableSet["Node"]),
			},
			"Element": {
				Name:       "Element",
				Kind:       ast.Interface,
				Interfaces: []string{"Node"},
				Fields:     []*ast.FieldDefinition{{Name: "id"}, {Name: "name"}},
				Directives: directives(hasDirective("Element")),
			},
			"Metal": {
				Name:       "Metal",
				Kind:       ast.Interface,
				Interfaces: []string{"Element"},
				Fields:     []*ast.FieldDefinition{{Name: "id"}, {Name: "name"}, {Name: "atomicNumber"}},
				Directives: directives(hasDirective("Metal")),
			},
		},
	}
}

// createABCChainSchema creates an A->B->C hierarchy.
// embeddable specifies which interfaces should have the goEmbedInterface directive.
func createABCChainSchema(embeddable ...string) *ast.Schema {
	embeddableSet := make(map[string]bool)
	for _, name := range embeddable {
		embeddableSet[name] = true
	}

	return &ast.Schema{
		Types: map[string]*ast.Definition{
			"A": {
				Name:       "A",
				Kind:       ast.Interface,
				Interfaces: []string{},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}},
				Directives: directives(embeddableSet["A"]),
			},
			"B": {
				Name:       "B",
				Kind:       ast.Interface,
				Interfaces: []string{"A"},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}, {Name: "fieldB"}},
				Directives: directives(embeddableSet["B"]),
			},
			"C": {
				Name:       "C",
				Kind:       ast.Interface,
				Interfaces: []string{"B"},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}, {Name: "fieldB"}, {Name: "fieldC"}},
				Directives: directives(embeddableSet["C"]),
			},
		},
	}
}

// createABCDChainSchema creates an A->B->C->D hierarchy.
// embeddable specifies which interfaces should have the goEmbedInterface directive.
func createABCDChainSchema(embeddable ...string) *ast.Schema {
	embeddableSet := make(map[string]bool)
	for _, name := range embeddable {
		embeddableSet[name] = true
	}

	return &ast.Schema{
		Types: map[string]*ast.Definition{
			"A": {
				Name:       "A",
				Kind:       ast.Interface,
				Interfaces: []string{},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}},
				Directives: directives(embeddableSet["A"]),
			},
			"B": {
				Name:       "B",
				Kind:       ast.Interface,
				Interfaces: []string{"A"},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}, {Name: "fieldB"}},
				Directives: directives(embeddableSet["B"]),
			},
			"C": {
				Name:       "C",
				Kind:       ast.Interface,
				Interfaces: []string{"B"},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}, {Name: "fieldB"}, {Name: "fieldC"}},
				Directives: directives(embeddableSet["C"]),
			},
			"D": {
				Name:       "D",
				Kind:       ast.Interface,
				Interfaces: []string{"C"},
				Fields:     []*ast.FieldDefinition{{Name: "fieldA"}, {Name: "fieldB"}, {Name: "fieldC"}, {Name: "fieldD"}},
				Directives: directives(embeddableSet["D"]),
			},
		},
	}
}

func directives(condition bool) ast.DirectiveList {
	if condition {
		return ast.DirectiveList{{Name: "goEmbedInterface"}}
	}
	return ast.DirectiveList{}
}
