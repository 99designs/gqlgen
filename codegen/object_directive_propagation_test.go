package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/internal/code"
)

// TestObjectDirectivePropagation guards against the regression introduced in
// PR #4029 (v0.17.87), where directives declared with both OBJECT and
// INPUT_OBJECT locations were stripped from every field of OBJECT/INTERFACE
// types — even though they were applied at the OBJECT level. The INPUT_OBJECT
// guard from issue #2281 must still hold.
func TestObjectDirectivePropagation(t *testing.T) {
	const schemaSrc = `
		schema { query: Query, mutation: Mutation }

		directive @auth(roles: [String!]!) on
			OBJECT | INTERFACE | FIELD_DEFINITION | INPUT_FIELD_DEFINITION | INPUT_OBJECT

		type Query {
			secret(id: ID!): Secret
			node(id: ID!): Node
		}

		type Mutation {
			createSecret(input: CreateSecretInput!): Secret
		}

		"OBJECT-level directive must propagate to every field resolver."
		type Secret @auth(roles: ["ADMIN"]) {
			id: ID!
			value: String!
		}

		"INTERFACE-level directive must also propagate."
		interface Node @auth(roles: ["VIEWER"]) {
			id: ID!
		}

		"INPUT_OBJECT-level directive must NOT propagate (#2281)."
		input CreateSecretInput @auth(roles: ["ADMIN"]) {
			name: String!
		}
	`

	b := newTestBuilder(t, schemaSrc)

	t.Run("OBJECT directive propagates to every field", func(t *testing.T) {
		obj, err := b.buildObject(b.Schema.Types["Secret"])
		require.NoError(t, err)
		for _, f := range obj.Fields {
			require.True(t,
				hasDirective(f.Directives, "auth"),
				"Secret.%s should inherit @auth from its OBJECT type", f.Name)
		}
	})

	t.Run("INTERFACE directive propagates to every field", func(t *testing.T) {
		obj, err := b.buildObject(b.Schema.Types["Node"])
		require.NoError(t, err)
		for _, f := range obj.Fields {
			require.True(t,
				hasDirective(f.Directives, "auth"),
				"Node.%s should inherit @auth from its INTERFACE type", f.Name)
		}
	})

	t.Run("INPUT_OBJECT directive does not leak to fields referencing the type (#2281)", func(t *testing.T) {
		mutation, err := b.buildObject(b.Schema.Mutation)
		require.NoError(t, err)
		var createSecret *Field
		for _, f := range mutation.Fields {
			if f.Name == "createSecret" {
				createSecret = f
				break
			}
		}
		require.NotNil(t, createSecret, "createSecret field should exist")
		require.False(t,
			hasDirective(createSecret.Directives, "auth"),
			"Mutation.createSecret should not inherit @auth from its INPUT_OBJECT argument type")
	})
}

func hasDirective(dirs []*Directive, name string) bool {
	for _, d := range dirs {
		if d.Name == name {
			return true
		}
	}
	return false
}

func newTestBuilder(t *testing.T, schemaSrc string) *builder {
	t.Helper()
	cfg := &config.Config{
		Exec: config.ExecConfig{
			Layout:   config.ExecLayoutSingleFile,
			Filename: "generated.go",
			Package:  "generated",
		},
		Models: config.TypeMap{
			"Boolean":           {Model: config.StringList{"github.com/99designs/gqlgen/graphql.Boolean"}},
			"Float":             {Model: config.StringList{"github.com/99designs/gqlgen/graphql.Float"}},
			"ID":                {Model: config.StringList{"github.com/99designs/gqlgen/graphql.ID"}},
			"Int":               {Model: config.StringList{"github.com/99designs/gqlgen/graphql.Int"}},
			"String":            {Model: config.StringList{"github.com/99designs/gqlgen/graphql.String"}},
			"Secret":            {Model: config.StringList{"map[string]any"}},
			"Node":              {Model: config.StringList{"map[string]any"}},
			"CreateSecretInput": {Model: config.StringList{"map[string]any"}},
		},
		Directives: map[string]config.DirectiveConfig{},
		Packages:   code.NewPackages(),
	}
	cfg.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: schemaSrc})

	b := &builder{Config: cfg, Schema: cfg.Schema}
	b.Binder = b.Config.NewBinder()

	var err error
	b.Directives, err = b.buildDirectives()
	require.NoError(t, err)
	return b
}
