package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestDirectiveParsing(t *testing.T) {
	t.Run("autoBindGetterHaser argument in goField", func(t *testing.T) {
		cfg := Config{
			Models:     TypeMap{},
			Directives: map[string]DirectiveConfig{},
		}

		cfg.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: `
			directive @goField(
				forceResolver: Boolean
				name: String
				omittable: Boolean
				autoBindGetterHaser: Boolean
			) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION

			type MyType {
				field1: String @goField(autoBindGetterHaser: true)
				field2: String @goField(autoBindGetterHaser: false)
				field3: String
			}
		`})

		err := cfg.injectTypesFromSchema()
		require.NoError(t, err)

		field1 := cfg.Models["MyType"].Fields["field1"]
		require.NotNil(t, field1.AutoBindGetterHaser)
		require.True(t, *field1.AutoBindGetterHaser)

		field2 := cfg.Models["MyType"].Fields["field2"]
		require.NotNil(t, field2.AutoBindGetterHaser)
		require.False(t, *field2.AutoBindGetterHaser)

		field3 := cfg.Models["MyType"].Fields["field3"]
		require.Nil(t, field3.AutoBindGetterHaser)
	})

	t.Run("batch argument in goField", func(t *testing.T) {
		cfg := Config{
			Models:     TypeMap{},
			Directives: map[string]DirectiveConfig{},
		}

		cfg.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: `
			directive @goField(batch: Boolean) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION

			type MyType {
				batchNull: String @goField(batch: null)
				batchTrue: String @goField(batch: true)
				batchFalse: String @goField(batch: false)
				noBatch: String
			}
		`})

		err := cfg.injectTypesFromSchema()
		require.NoError(t, err)

		m := cfg.Models["MyType"]
		require.Nil(t, m.Fields["batchNull"].Batch)
		require.NotNil(t, m.Fields["batchTrue"].Batch)
		require.True(t, *m.Fields["batchTrue"].Batch)
		require.NotNil(t, m.Fields["batchFalse"].Batch)
		require.False(t, *m.Fields["batchFalse"].Batch)
		_, ok := m.Fields["noBatch"]
		require.False(t, ok)
	})

	t.Run("resolver.batch config enables batch for all fields", func(t *testing.T) {
		cfg := Config{
			Models:     TypeMap{},
			Directives: map[string]DirectiveConfig{},
			Resolver: ResolverConfig{
				Batch: ResolverBatchConfig{Enabled: true},
			},
		}

		cfg.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: `
			directive @goField(batch: Boolean) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION

			type MyType {
				batchTrue: String @goField(batch: true)
				batchFalse: String @goField(batch: false)
				noBatch: String
			}
		`})

		err := cfg.injectTypesFromSchema()
		require.NoError(t, err)

		m := cfg.Models["MyType"]
		require.True(t, *m.Fields["batchTrue"].Batch)
		require.False(t, *m.Fields["batchFalse"].Batch)
		_, ok := m.Fields["noBatch"]
		require.False(t, ok)
	})

	t.Run("batch priority: config, models yaml, goField directive", func(t *testing.T) {
		falseVal := false
		trueVal := true
		cfg := Config{
			Resolver: ResolverConfig{
				Batch: ResolverBatchConfig{Enabled: true},
			},
			Models: TypeMap{
				"MyType": {
					Fields: map[string]TypeMapField{
						"fromYaml":             {Batch: &falseVal},
						"fromYamlAndDirective": {Batch: &trueVal},
					},
				},
			},
			Directives: map[string]DirectiveConfig{},
		}

		cfg.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: `
			directive @goField(batch: Boolean) on INPUT_FIELD_DEFINITION | FIELD_DEFINITION

			type MyType {
				fromYaml: String
				fromYamlAndDirective: String @goField(batch: false)
				fromConfigOnly: String
			}
		`})

		err := cfg.injectTypesFromSchema()
		require.NoError(t, err)

		m := cfg.Models["MyType"]
		require.False(t, *m.Fields["fromYaml"].Batch)
		require.False(t, *m.Fields["fromYamlAndDirective"].Batch)
		_, ok := m.Fields["fromConfigOnly"]
		require.False(t, ok)
	})
}
