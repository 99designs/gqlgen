package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestBuildDependencyGraph(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"User": {
				Name: "User",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "id", Type: ast.NamedType("ID", nil)},
					{Name: "posts", Type: ast.ListType(ast.NamedType("Post", nil), nil)},
				},
				Position: &ast.Position{Src: &ast.Source{Name: "user.graphqls"}},
			},
			"Post": {
				Name: "Post",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "id", Type: ast.NamedType("ID", nil)},
					{Name: "author", Type: ast.NamedType("User", nil)},
					{Name: "comments", Type: ast.ListType(ast.NamedType("Comment", nil), nil)},
				},
				Position: &ast.Position{Src: &ast.Source{Name: "post.graphqls"}},
			},
			"Comment": {
				Name: "Comment",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{Name: "id", Type: ast.NamedType("ID", nil)},
					{Name: "text", Type: ast.NamedType("String", nil)},
				},
				Position: &ast.Position{Src: &ast.Source{Name: "comment.graphqls"}},
			},
			"String": {Name: "String", Kind: ast.Scalar, BuiltIn: true},
			"ID":     {Name: "ID", Kind: ast.Scalar, BuiltIn: true},
		},
	}

	graph := BuildDependencyGraph(schema)

	// Verify type-to-schema mapping
	require.Equal(t, "user.graphqls", graph.TypeToSchema["User"])
	require.Equal(t, "post.graphqls", graph.TypeToSchema["Post"])
	require.Equal(t, "comment.graphqls", graph.TypeToSchema["Comment"])

	// Verify schema-to-types mapping
	require.ElementsMatch(t, []string{"User"}, graph.SchemaToTypes["user.graphqls"])
	require.ElementsMatch(t, []string{"Post"}, graph.SchemaToTypes["post.graphqls"])
	require.ElementsMatch(t, []string{"Comment"}, graph.SchemaToTypes["comment.graphqls"])

	// Verify type dependencies (User -> Post, Post -> User + Comment)
	require.Contains(t, graph.TypeDependencies["User"], "Post")
	require.Contains(t, graph.TypeDependencies["Post"], "User")
	require.Contains(t, graph.TypeDependencies["Post"], "Comment")

	// Verify schema dependencies (cross-schema references)
	require.True(t, graph.SchemaDependencies["user.graphqls"]["post.graphqls"])
	require.True(t, graph.SchemaDependencies["post.graphqls"]["user.graphqls"])
	require.True(t, graph.SchemaDependencies["post.graphqls"]["comment.graphqls"])

	// Verify built-in types are not included
	require.Empty(t, graph.TypeToSchema["String"])
	require.Empty(t, graph.TypeToSchema["ID"])
}

func TestBuildDependencyGraph_Interfaces(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"Node": {
				Name:     "Node",
				Kind:     ast.Interface,
				Fields:   ast.FieldList{{Name: "id", Type: ast.NamedType("ID", nil)}},
				Position: &ast.Position{Src: &ast.Source{Name: "node.graphqls"}},
			},
			"User": {
				Name:       "User",
				Kind:       ast.Object,
				Interfaces: []string{"Node"},
				Fields:     ast.FieldList{{Name: "id", Type: ast.NamedType("ID", nil)}},
				Position:   &ast.Position{Src: &ast.Source{Name: "user.graphqls"}},
			},
			"ID": {Name: "ID", Kind: ast.Scalar, BuiltIn: true},
		},
	}

	graph := BuildDependencyGraph(schema)

	// User implements Node, so User depends on Node
	require.Contains(t, graph.TypeDependencies["User"], "Node")
	require.True(t, graph.SchemaDependencies["user.graphqls"]["node.graphqls"])
}

func TestBuildDependencyGraph_Union(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"SearchResult": {
				Name:     "SearchResult",
				Kind:     ast.Union,
				Types:    []string{"User", "Post"},
				Position: &ast.Position{Src: &ast.Source{Name: "search.graphqls"}},
			},
			"User": {
				Name:     "User",
				Kind:     ast.Object,
				Fields:   ast.FieldList{{Name: "name", Type: ast.NamedType("String", nil)}},
				Position: &ast.Position{Src: &ast.Source{Name: "user.graphqls"}},
			},
			"Post": {
				Name:     "Post",
				Kind:     ast.Object,
				Fields:   ast.FieldList{{Name: "title", Type: ast.NamedType("String", nil)}},
				Position: &ast.Position{Src: &ast.Source{Name: "post.graphqls"}},
			},
			"String": {Name: "String", Kind: ast.Scalar, BuiltIn: true},
		},
	}

	graph := BuildDependencyGraph(schema)

	// SearchResult union depends on User and Post
	require.Contains(t, graph.TypeDependencies["SearchResult"], "User")
	require.Contains(t, graph.TypeDependencies["SearchResult"], "Post")
	require.True(t, graph.SchemaDependencies["search.graphqls"]["user.graphqls"])
	require.True(t, graph.SchemaDependencies["search.graphqls"]["post.graphqls"])
}

func TestBuildDependencyGraph_FieldArguments(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"Query": {
				Name: "Query",
				Kind: ast.Object,
				Fields: ast.FieldList{
					{
						Name: "user",
						Type: ast.NamedType("User", nil),
						Arguments: ast.ArgumentDefinitionList{
							{Name: "filter", Type: ast.NamedType("UserFilter", nil)},
						},
					},
				},
				Position: &ast.Position{Src: &ast.Source{Name: "query.graphqls"}},
			},
			"User": {
				Name:     "User",
				Kind:     ast.Object,
				Fields:   ast.FieldList{{Name: "id", Type: ast.NamedType("ID", nil)}},
				Position: &ast.Position{Src: &ast.Source{Name: "user.graphqls"}},
			},
			"UserFilter": {
				Name:     "UserFilter",
				Kind:     ast.InputObject,
				Fields:   ast.FieldList{{Name: "name", Type: ast.NamedType("String", nil)}},
				Position: &ast.Position{Src: &ast.Source{Name: "filter.graphqls"}},
			},
			"ID":     {Name: "ID", Kind: ast.Scalar, BuiltIn: true},
			"String": {Name: "String", Kind: ast.Scalar, BuiltIn: true},
		},
	}

	graph := BuildDependencyGraph(schema)

	// Query depends on User (return type) and UserFilter (argument type)
	require.Contains(t, graph.TypeDependencies["Query"], "User")
	require.Contains(t, graph.TypeDependencies["Query"], "UserFilter")
	require.True(t, graph.SchemaDependencies["query.graphqls"]["user.graphqls"])
	require.True(t, graph.SchemaDependencies["query.graphqls"]["filter.graphqls"])
}

func TestGetAffectedSchemas(t *testing.T) {
	// Build a dependency chain: C -> B -> A (C depends on B, B depends on A)
	graph := &DependencyGraph{
		SchemaToTypes: map[string][]string{
			"a.graphqls": {"TypeA"},
			"b.graphqls": {"TypeB"},
			"c.graphqls": {"TypeC"},
			"d.graphqls": {"TypeD"},
		},
		TypeToSchema: map[string]string{
			"TypeA": "a.graphqls",
			"TypeB": "b.graphqls",
			"TypeC": "c.graphqls",
			"TypeD": "d.graphqls",
		},
		SchemaDependencies: map[string]map[string]bool{
			"a.graphqls": {},
			"b.graphqls": {"a.graphqls": true}, // B depends on A
			"c.graphqls": {"b.graphqls": true}, // C depends on B
			"d.graphqls": {},                   // D is independent
		},
	}

	tests := []struct {
		name     string
		changed  []string
		expected []string
	}{
		{
			name:     "change at root propagates transitively",
			changed:  []string{"a.graphqls"},
			expected: []string{"a.graphqls", "b.graphqls", "c.graphqls"},
		},
		{
			name:     "change at middle propagates to dependents only",
			changed:  []string{"b.graphqls"},
			expected: []string{"b.graphqls", "c.graphqls"},
		},
		{
			name:     "change at leaf affects only itself",
			changed:  []string{"c.graphqls"},
			expected: []string{"c.graphqls"},
		},
		{
			name:     "independent schema affects only itself",
			changed:  []string{"d.graphqls"},
			expected: []string{"d.graphqls"},
		},
		{
			name:     "multiple changes are merged",
			changed:  []string{"a.graphqls", "d.graphqls"},
			expected: []string{"a.graphqls", "b.graphqls", "c.graphqls", "d.graphqls"},
		},
		{
			name:     "empty input returns empty",
			changed:  []string{},
			expected: []string{},
		},
		{
			name:     "unknown schema is still included",
			changed:  []string{"unknown.graphqls"},
			expected: []string{"unknown.graphqls"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected := graph.GetAffectedSchemas(tt.changed)
			require.ElementsMatch(t, tt.expected, affected)
		})
	}
}

func TestGetAffectedSchemas_CircularDependency(t *testing.T) {
	// A -> B -> C -> A (circular)
	graph := &DependencyGraph{
		SchemaToTypes: map[string][]string{
			"a.graphqls": {"TypeA"},
			"b.graphqls": {"TypeB"},
			"c.graphqls": {"TypeC"},
		},
		SchemaDependencies: map[string]map[string]bool{
			"a.graphqls": {"c.graphqls": true}, // A depends on C
			"b.graphqls": {"a.graphqls": true}, // B depends on A
			"c.graphqls": {"b.graphqls": true}, // C depends on B
		},
	}

	// Changing any one should affect all three
	affected := graph.GetAffectedSchemas([]string{"a.graphqls"})
	require.ElementsMatch(t, []string{"a.graphqls", "b.graphqls", "c.graphqls"}, affected)

	affected = graph.GetAffectedSchemas([]string{"b.graphqls"})
	require.ElementsMatch(t, []string{"a.graphqls", "b.graphqls", "c.graphqls"}, affected)
}

func TestGetTypesForSchemas(t *testing.T) {
	graph := &DependencyGraph{
		SchemaToTypes: map[string][]string{
			"a.graphqls": {"TypeA1", "TypeA2"},
			"b.graphqls": {"TypeB"},
			"c.graphqls": {},
		},
	}

	tests := []struct {
		name     string
		schemas  []string
		expected map[string]bool
	}{
		{
			name:     "single schema with multiple types",
			schemas:  []string{"a.graphqls"},
			expected: map[string]bool{"TypeA1": true, "TypeA2": true},
		},
		{
			name:     "multiple schemas",
			schemas:  []string{"a.graphqls", "b.graphqls"},
			expected: map[string]bool{"TypeA1": true, "TypeA2": true, "TypeB": true},
		},
		{
			name:     "empty schema",
			schemas:  []string{"c.graphqls"},
			expected: map[string]bool{},
		},
		{
			name:     "unknown schema",
			schemas:  []string{"unknown.graphqls"},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			types := graph.GetTypesForSchemas(tt.schemas)
			require.Equal(t, tt.expected, types)
		})
	}
}

func TestIsBuiltInScalar(t *testing.T) {
	builtIns := []string{"Int", "Float", "String", "Boolean", "ID"}
	for _, name := range builtIns {
		require.True(t, isBuiltInScalar(name), "%s should be a built-in scalar", name)
	}

	nonBuiltIns := []string{"User", "Post", "DateTime", "JSON", ""}
	for _, name := range nonBuiltIns {
		require.False(t, isBuiltInScalar(name), "%s should not be a built-in scalar", name)
	}
}
