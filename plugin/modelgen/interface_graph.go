package modelgen

import (
	"errors"

	"github.com/vektah/gqlparser/v2/ast"
)

// interfaceGraph tracks interface implementation relationships.
type interfaceGraph struct {
	schema           *ast.Schema
	parentInterfaces map[string][]string // interface -> interfaces it implements
	childInterfaces  map[string][]string // interface -> interfaces that implement it
}

func newInterfaceGraph(schema *ast.Schema) *interfaceGraph {
	g := &interfaceGraph{
		schema:           schema,
		parentInterfaces: make(map[string][]string),
		childInterfaces:  make(map[string][]string),
	}

	for _, schemaType := range schema.Types {
		if schemaType.Kind != ast.Interface {
			continue
		}

		if len(schemaType.Interfaces) == 0 {
			g.parentInterfaces[schemaType.Name] = []string{}
		} else {
			g.parentInterfaces[schemaType.Name] = append([]string{}, schemaType.Interfaces...)
			for _, parent := range schemaType.Interfaces {
				g.childInterfaces[parent] = append(g.childInterfaces[parent], schemaType.Name)
			}
		}
	}

	return g
}

// topologicalSort returns interfaces ordered with parents before children.
func (g *interfaceGraph) topologicalSort(interfaces []string) ([]string, error) {
	inDegree := make(map[string]int)
	for _, iface := range interfaces {
		inDegree[iface] = len(g.parentInterfaces[iface])
	}

	var queue []string
	for _, iface := range interfaces {
		if inDegree[iface] == 0 {
			queue = append(queue, iface)
		}
	}

	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, child := range g.childInterfaces[current] {
			if _, exists := inDegree[child]; exists {
				inDegree[child]--
				if inDegree[child] == 0 {
					queue = append(queue, child)
				}
			}
		}
	}

	if len(result) != len(interfaces) {
		return nil, errors.New("cycle detected in interface implementations")
	}

	return result, nil
}

// getInterfaceOwnFields returns only the fields that are not inherited from parent interfaces.
func (g *interfaceGraph) getInterfaceOwnFields(interfaceName string) []*ast.FieldDefinition {
	schemaInterface := g.schema.Types[interfaceName]
	if schemaInterface == nil || schemaInterface.Kind != ast.Interface {
		return nil
	}

	parents := g.parentInterfaces[interfaceName]
	if len(parents) == 0 {
		return schemaInterface.Fields
	}

	parentFieldNames := make(map[string]bool)
	for _, parentName := range parents {
		parentInterface := g.schema.Types[parentName]
		if parentInterface == nil {
			continue
		}
		for _, field := range parentInterface.Fields {
			parentFieldNames[field.Name] = true
		}
	}

	ownFields := []*ast.FieldDefinition{}
	for _, field := range schemaInterface.Fields {
		if !parentFieldNames[field.Name] {
			ownFields = append(ownFields, field)
		}
	}

	return ownFields
}
