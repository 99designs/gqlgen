package codegen

import (
	"github.com/vektah/gqlparser/v2/ast"
)

// DependencyGraph tracks type definitions and their cross-schema dependencies.
// This enables incremental generation by computing which schemas are affected
// when a subset of schema files change.
type DependencyGraph struct {
	// SchemaToTypes maps schema file path -> type names defined in that file
	SchemaToTypes map[string][]string

	// TypeToSchema maps type name -> schema file path where it's defined
	TypeToSchema map[string]string

	// TypeDependencies maps type name -> type names it references
	TypeDependencies map[string][]string

	// SchemaDependencies maps schema file -> schema files it depends on
	SchemaDependencies map[string]map[string]bool
}

// NewDependencyGraph creates a new empty dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		SchemaToTypes:      make(map[string][]string),
		TypeToSchema:       make(map[string]string),
		TypeDependencies:   make(map[string][]string),
		SchemaDependencies: make(map[string]map[string]bool),
	}
}

// BuildDependencyGraph constructs a dependency graph from the parsed schema
func BuildDependencyGraph(schema *ast.Schema) *DependencyGraph {
	g := NewDependencyGraph()

	// First pass: map each type to its source schema file
	for _, typ := range schema.Types {
		if typ.BuiltIn || typ.Position == nil || typ.Position.Src == nil {
			continue
		}
		schemaFile := typ.Position.Src.Name
		g.TypeToSchema[typ.Name] = schemaFile
		g.SchemaToTypes[schemaFile] = append(g.SchemaToTypes[schemaFile], typ.Name)
	}

	// Second pass: find type dependencies
	for _, typ := range schema.Types {
		if typ.BuiltIn || typ.Position == nil || typ.Position.Src == nil {
			continue
		}
		if deps := g.extractTypeDependencies(typ); len(deps) > 0 {
			g.TypeDependencies[typ.Name] = deps
		}
	}

	// Third pass: build schema-level dependencies
	for schemaFile, types := range g.SchemaToTypes {
		g.SchemaDependencies[schemaFile] = make(map[string]bool)
		for _, typeName := range types {
			for _, depType := range g.TypeDependencies[typeName] {
				if depSchema, ok := g.TypeToSchema[depType]; ok && depSchema != schemaFile {
					g.SchemaDependencies[schemaFile][depSchema] = true
				}
			}
		}
	}

	return g
}

// extractTypeDependencies finds all types referenced by the given type definition
func (g *DependencyGraph) extractTypeDependencies(typ *ast.Definition) []string {
	deps := make(map[string]bool)

	for _, iface := range typ.Interfaces {
		deps[iface] = true
	}
	for _, unionType := range typ.Types {
		deps[unionType] = true
	}
	for _, field := range typ.Fields {
		g.extractTypeRef(field.Type, deps)
		for _, arg := range field.Arguments {
			g.extractTypeRef(arg.Type, deps)
		}
	}

	result := make([]string, 0, len(deps))
	for dep := range deps {
		if !isBuiltInScalar(dep) {
			result = append(result, dep)
		}
	}
	return result
}

func (g *DependencyGraph) extractTypeRef(t *ast.Type, deps map[string]bool) {
	if t == nil {
		return
	}
	if t.Elem != nil {
		g.extractTypeRef(t.Elem, deps)
	} else {
		deps[t.NamedType] = true
	}
}

func isBuiltInScalar(name string) bool {
	switch name {
	case "Int", "Float", "String", "Boolean", "ID":
		return true
	}
	return false
}

// GetAffectedSchemas returns all schema files that need regeneration when
// the given schemas have changed. Computes transitive closure of dependencies.
func (g *DependencyGraph) GetAffectedSchemas(changedSchemas []string) []string {
	affected := make(map[string]bool)
	for _, s := range changedSchemas {
		affected[s] = true
	}

	// Iterate until fixed point (transitive closure)
	changed := true
	for changed {
		changed = false
		for schema := range affected {
			for otherSchema, deps := range g.SchemaDependencies {
				if !affected[otherSchema] && deps[schema] {
					affected[otherSchema] = true
					changed = true
				}
			}
		}
	}

	result := make([]string, 0, len(affected))
	for s := range affected {
		result = append(result, s)
	}
	return result
}

// GetTypesForSchemas returns all type names defined in the given schema files
func (g *DependencyGraph) GetTypesForSchemas(schemas []string) map[string]bool {
	types := make(map[string]bool)
	for _, schema := range schemas {
		for _, typeName := range g.SchemaToTypes[schema] {
			types[typeName] = true
		}
	}
	return types
}
