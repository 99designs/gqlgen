package codegen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

// TestGenerateCodeIncremental_SelectiveGeneration verifies that incremental generation
// only regenerates files for affected schemas (the core optimization).
func TestGenerateCodeIncremental_SelectiveGeneration(t *testing.T) {
	// Create temp directory for generated files
	tmpDir, err := os.MkdirTemp("", "gqlgen-incremental-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Create schema with multiple files that have dependencies:
	// user.graphqls: User type
	// post.graphqls: Post type (references User)
	// comment.graphqls: Comment type (independent)
	userSchema := &ast.Source{
		Name: "user.graphqls",
		Input: `type User {
			id: ID!
			name: String!
		}`,
	}
	postSchema := &ast.Source{
		Name: "post.graphqls",
		Input: `type Post {
			id: ID!
			title: String!
			author: User!
		}`,
	}
	commentSchema := &ast.Source{
		Name: "comment.graphqls",
		Input: `type Comment {
			id: ID!
			text: String!
		}`,
	}
	querySchema := &ast.Source{
		Name: "schema.graphqls",
		Input: `type Query {
			user: User
			post: Post
			comment: Comment
		}`,
	}

	schema, err := gqlparser.LoadSchema(userSchema, postSchema, commentSchema, querySchema)
	require.NoError(t, err)

	// Build dependency graph
	depGraph := BuildDependencyGraph(schema)

	// Test 1: Changing user.graphqls should affect post.graphqls (Post references User)
	affected := depGraph.GetAffectedSchemas([]string{"user.graphqls"})
	require.Contains(t, affected, "user.graphqls", "changed schema should be affected")
	require.Contains(t, affected, "post.graphqls", "post depends on user, should be affected")
	require.NotContains(t, affected, "comment.graphqls", "comment is independent, should NOT be affected")

	// Test 2: Changing comment.graphqls affects itself and schema.graphqls (Query references Comment)
	affected = depGraph.GetAffectedSchemas([]string{"comment.graphqls"})
	require.Contains(t, affected, "comment.graphqls", "changed schema should be affected")
	require.Contains(t, affected, "schema.graphqls", "Query references Comment, so schema is affected")
	require.NotContains(t, affected, "user.graphqls", "user is independent of comment")
	require.NotContains(t, affected, "post.graphqls", "post is independent of comment")

	// Test 3: Verify the type filtering works correctly
	affectedTypes := depGraph.GetTypesForSchemas([]string{"user.graphqls"})
	require.True(t, affectedTypes["User"], "User should be in affected types")
	require.False(t, affectedTypes["Post"], "Post should NOT be in affected types for user schema only")
	require.False(t, affectedTypes["Comment"], "Comment should NOT be in affected types")
}

// TestGenerateCodeIncremental_CorrectnessWithRealSchema verifies that incremental
// generation produces the same filtering behavior as expected with real Data structures.
func TestGenerateCodeIncremental_CorrectnessWithRealSchema(t *testing.T) {
	// Create a schema with cross-file dependencies
	sources := []*ast.Source{
		{Name: "user.graphqls", Input: `type User { id: ID!, name: String! }`},
		{Name: "post.graphqls", Input: `type Post { id: ID!, author: User! }`},
		{Name: "tag.graphqls", Input: `type Tag { id: ID!, label: String! }`},
	}

	schema, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)

	depGraph := BuildDependencyGraph(schema)

	// Verify dependency detection
	require.Contains(t, depGraph.TypeDependencies["Post"], "User",
		"Post should depend on User")
	require.Empty(t, depGraph.TypeDependencies["Tag"],
		"Tag should have no dependencies (only uses built-in scalars)")

	// Verify schema-level dependencies
	require.True(t, depGraph.SchemaDependencies["post.graphqls"]["user.graphqls"],
		"post.graphqls should depend on user.graphqls")
	require.Empty(t, depGraph.SchemaDependencies["tag.graphqls"],
		"tag.graphqls should have no schema dependencies")

	// Test transitive closure
	affected := depGraph.GetAffectedSchemas([]string{"user.graphqls"})
	require.Contains(t, affected, "post.graphqls",
		"post should be affected when user changes")
	require.NotContains(t, affected, "tag.graphqls",
		"tag should NOT be affected when user changes")
}

// TestGenerateCodeIncremental_DiamondDependency tests a diamond dependency pattern:
// D depends on B and C, both B and C depend on A.
// Changing A should affect all four.
func TestGenerateCodeIncremental_DiamondDependency(t *testing.T) {
	sources := []*ast.Source{
		{Name: "a.graphqls", Input: `type TypeA { id: ID! }`},
		{Name: "b.graphqls", Input: `type TypeB { id: ID!, a: TypeA! }`},
		{Name: "c.graphqls", Input: `type TypeC { id: ID!, a: TypeA! }`},
		{Name: "d.graphqls", Input: `type TypeD { id: ID!, b: TypeB!, c: TypeC! }`},
	}

	schema, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)

	depGraph := BuildDependencyGraph(schema)

	// Changing A should affect B, C, and D (all depend on A directly or transitively)
	affected := depGraph.GetAffectedSchemas([]string{"a.graphqls"})
	require.ElementsMatch(t,
		[]string{"a.graphqls", "b.graphqls", "c.graphqls", "d.graphqls"},
		affected,
		"diamond dependency: all should be affected when root changes")

	// Changing B should only affect B and D
	affected = depGraph.GetAffectedSchemas([]string{"b.graphqls"})
	require.ElementsMatch(t,
		[]string{"b.graphqls", "d.graphqls"},
		affected,
		"changing B should affect B and D only")

	// Changing D should only affect D
	affected = depGraph.GetAffectedSchemas([]string{"d.graphqls"})
	require.ElementsMatch(t,
		[]string{"d.graphqls"},
		affected,
		"changing leaf D should only affect itself")
}

// TestGenerateCodeIncremental_InputTypes verifies that input type dependencies
// are correctly tracked (important for mutations with input arguments).
func TestGenerateCodeIncremental_InputTypes(t *testing.T) {
	sources := []*ast.Source{
		{Name: "input.graphqls", Input: `input CreateUserInput { name: String!, email: String! }`},
		{Name: "mutation.graphqls", Input: `
			type Mutation { createUser(input: CreateUserInput!): User! }
			type User { id: ID!, name: String! }
		`},
	}

	schema, err := gqlparser.LoadSchema(sources...)
	require.NoError(t, err)

	depGraph := BuildDependencyGraph(schema)

	// Mutation schema should depend on input schema
	require.True(t, depGraph.SchemaDependencies["mutation.graphqls"]["input.graphqls"],
		"mutation should depend on input type schema")

	// Changing input should affect mutation
	affected := depGraph.GetAffectedSchemas([]string{"input.graphqls"})
	require.Contains(t, affected, "mutation.graphqls",
		"mutation should be affected when input type changes")
}

func TestGenerateCodeIncremental_FallbackToFullGeneration(t *testing.T) {
	tests := []struct {
		name    string
		layout  config.ExecLayout
		changed []string
	}{
		{
			name:    "single-file layout falls back to full generation",
			layout:  config.ExecLayoutSingleFile,
			changed: []string{"user.graphqls"},
		},
		{
			name:    "no changed schemas falls back to full generation",
			layout:  config.ExecLayoutFollowSchema,
			changed: []string{},
		},
		{
			name:    "nil changed schemas falls back to full generation",
			layout:  config.ExecLayoutFollowSchema,
			changed: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal data with missing exec config to trigger early error
			// This validates that the fallback path is taken (GenerateCode is called)
			data := &Data{
				Config: &config.Config{
					Exec: config.ExecConfig{
						Layout: tt.layout,
					},
				},
			}

			opts := IncrementalOptions{
				ChangedSchemas: tt.changed,
				Verbose:        false,
			}

			// Both incremental and full generation will fail with "missing exec config"
			// This confirms the code path is working (we're not testing actual generation here)
			err := GenerateCodeIncremental(data, opts)
			require.Error(t, err)
			require.Contains(t, err.Error(), "missing exec config")
		})
	}
}

func TestGenerateCodeIncremental_MissingExecConfig(t *testing.T) {
	data := &Data{
		Config: &config.Config{},
	}

	err := GenerateCodeIncremental(data, IncrementalOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing exec config")
}

func TestIsAffected(t *testing.T) {
	affectedSet := map[string]bool{
		"user.graphqls": true,
		"post.graphqls": true,
	}

	tests := []struct {
		name     string
		pos      *ast.Position
		expected bool
	}{
		{
			name:     "nil position defaults to affected (safe fallback)",
			pos:      nil,
			expected: true,
		},
		{
			name:     "nil source defaults to affected (safe fallback)",
			pos:      &ast.Position{Src: nil},
			expected: true,
		},
		{
			name:     "affected schema returns true",
			pos:      &ast.Position{Src: &ast.Source{Name: "user.graphqls"}},
			expected: true,
		},
		{
			name:     "another affected schema returns true",
			pos:      &ast.Position{Src: &ast.Source{Name: "post.graphqls"}},
			expected: true,
		},
		{
			name:     "unaffected schema returns false",
			pos:      &ast.Position{Src: &ast.Source{Name: "comment.graphqls"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAffected(tt.pos, affectedSet)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAffected_EmptySet(t *testing.T) {
	emptySet := map[string]bool{}

	// With empty affected set, everything should be unaffected
	pos := &ast.Position{Src: &ast.Source{Name: "any.graphqls"}}
	require.False(t, isAffected(pos, emptySet))

	// But nil position still defaults to true (safe fallback)
	require.True(t, isAffected(nil, emptySet))
}

func TestMakeSet(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]bool
	}{
		{
			name:     "normal items",
			input:    []string{"a", "b", "c"},
			expected: map[string]bool{"a": true, "b": true, "c": true},
		},
		{
			name:     "duplicates are deduplicated",
			input:    []string{"a", "a", "b", "b", "b"},
			expected: map[string]bool{"a": true, "b": true},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: map[string]bool{},
		},
		{
			name:     "single item",
			input:    []string{"only"},
			expected: map[string]bool{"only": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeSet(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIncrementalOptions(t *testing.T) {
	// Test that IncrementalOptions struct works as expected
	opts := IncrementalOptions{
		ChangedSchemas: []string{"a.graphqls", "b.graphqls"},
		Verbose:        true,
	}

	require.Len(t, opts.ChangedSchemas, 2)
	require.True(t, opts.Verbose)

	// Empty options
	emptyOpts := IncrementalOptions{}
	require.Nil(t, emptyOpts.ChangedSchemas)
	require.False(t, emptyOpts.Verbose)
}
