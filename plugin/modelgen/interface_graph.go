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
// Only considers relationships between interfaces in the provided list.
func (g *interfaceGraph) topologicalSort(interfaces []string) ([]string, error) {
	interfaceSet := make(map[string]bool)
	for _, iface := range interfaces {
		interfaceSet[iface] = true
	}

	inDegree := make(map[string]int)
	for _, iface := range interfaces {
		count := 0
		for _, parent := range g.parentInterfaces[iface] {
			if interfaceSet[parent] {
				count++
			}
		}
		inDegree[iface] = count
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
			if interfaceSet[child] {
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

// embeddingInfo contains information about interface embedding relationships.
type embeddingInfo struct {
	Parents       []string               // embeddable parent interfaces with goEmbedInterface directive
	SkippedFields []*ast.FieldDefinition // fields from intermediate parents without the directive
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

// getEmbeddingInfo returns information about embeddable parent interfaces and fields
// from intermediate parents that don't have the goEmbedInterface directive.
func (g *interfaceGraph) getEmbeddingInfo(interfaceName string) embeddingInfo {
	info := embeddingInfo{
		Parents:       []string{},
		SkippedFields: []*ast.FieldDefinition{},
	}
	visited := make(map[string]bool)

	var walkParents func(name string)
	walkParents = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		parentDef := g.schema.Types[name]
		if parentDef == nil || parentDef.Kind != ast.Interface {
			return
		}

		// Check if this parent has the directive (is embeddable)
		if g.isEmbeddable(name) {
			info.Parents = append(info.Parents, name)
		} else {
			// Not embeddable - collect its fields and walk up
			info.SkippedFields = append(info.SkippedFields, g.getInterfaceOwnFields(name)...)
			for _, grandparent := range parentDef.Interfaces {
				walkParents(grandparent)
			}
		}
	}

	currentDef := g.schema.Types[interfaceName]
	if currentDef != nil {
		for _, parent := range currentDef.Interfaces {
			walkParents(parent)
		}
	}

	return info
}

// isEmbeddable returns true if the interface has the goEmbedInterface directive.
func (g *interfaceGraph) isEmbeddable(interfaceName string) bool {
	iface := g.schema.Types[interfaceName]
	if iface == nil || iface.Kind != ast.Interface {
		return false
	}
	return iface.Directives.ForName("goEmbedInterface") != nil
}
