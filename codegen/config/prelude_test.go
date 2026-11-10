package config

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

// TestPreludeDirectivesAreReviewed fails when a gqlparser upgrade adds or removes
// a built-in directive. CompleteConfig marks every prelude directive SkipRuntime,
// which only holds if gqlgen implements the directive somewhere other than
// generated code. Decide where the new one is handled before adding it here.
func TestPreludeDirectivesAreReviewed(t *testing.T) {
	reviewed := map[string]string{
		"skip":        "executor (graphql.shouldIncludeNode)",
		"include":     "executor (graphql.shouldIncludeNode)",
		"defer":       "executor (graphql.deferrable)",
		"deprecated":  "introspection (isDeprecated, deprecationReason, includeDeprecated)",
		"specifiedBy": "introspection (__Type.specifiedByURL)",
		"oneOf":       "introspection (__Type.isOneOf) and gqlparser validation",
	}

	var prelude []string
	for _, dir := range validator.PreludeDocument().Directives {
		prelude = append(prelude, dir.Name)
	}
	require.ElementsMatch(t, slices.Collect(maps.Keys(reviewed)), prelude,
		"the gqlparser prelude's directives changed; review how gqlgen handles them")
}

func TestCompleteConfigSkipsPreludeDirectivesAtRuntime(t *testing.T) {
	cases := map[string]struct {
		yaml        string
		wantRuntime map[string]bool
	}{
		"defaults": {
			yaml: "schema: [testdata/cfg/glob/**/*.graphql]\n",
		},
		"user configuration wins": {
			yaml: "schema: [testdata/cfg/glob/**/*.graphql]\n" +
				"directives:\n  deprecated:\n    skip_runtime: false\n",
			wantRuntime: map[string]bool{"deprecated": true},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, err := ReadConfig(strings.NewReader(tc.yaml))
			require.NoError(t, err)

			for _, dir := range validator.PreludeDocument().Directives {
				require.Contains(t, cfg.Directives, dir.Name)
				require.Equal(t, !tc.wantRuntime[dir.Name], cfg.Directives[dir.Name].SkipRuntime,
					"@%s SkipRuntime", dir.Name)
			}
		})
	}
}

func TestInjectBuiltinsBindsEveryPreludeType(t *testing.T) {
	cfg := Config{
		Models: TypeMap{},
		Schema: gqlparser.MustLoadSchema(&ast.Source{Input: "type Query { f: Int }"}),
	}
	cfg.injectBuiltins()

	for _, def := range validator.PreludeDocument().Definitions {
		require.NotEmpty(t, cfg.Models[def.Name].Model,
			"gqlparser's prelude declares %s %s, but injectBuiltins has no Go model for it",
			def.Kind, def.Name)
	}
}

func TestLoadSchemaGivesASynthesizedQueryIntrospectionFields(t *testing.T) {
	cfg, err := ReadConfig(strings.NewReader("schema: [testdata/cfg/glob/**/*.graphql]\n"))
	require.NoError(t, err)
	cfg.Sources = []*ast.Source{{Name: "schema.graphql", Input: "type Mutation { f: Int }"}}

	require.NoError(t, cfg.LoadSchema())

	require.Equal(t, validator.QueryIntrospectionFields(), cfg.Schema.Query.Fields)
}
