package introspection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestIncludeDeprecatedInputValues(t *testing.T) {
	schema := WrapSchema(gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: `
		directive @dir(old: Int @deprecated(reason: "gone"), current: Int) on FIELD
		input In {
			old: Int @deprecated(reason: "gone")
			current: Int
		}
		type Query {
			f(old: Int @deprecated(reason: "gone"), current: Int): Int
		}
	`}))

	var dir *Directive
	for _, d := range schema.Directives() {
		if d.Name == "dir" {
			dir = &d
		}
	}
	require.NotNil(t, dir)
	field := schema.QueryType().Fields(false)[0]
	var input *Type
	for _, typ := range schema.Types() {
		if *typ.Name() == "In" {
			input = &typ
		}
	}
	require.NotNil(t, input)

	sources := map[string]func(includeDeprecated bool) []InputValue{
		"__Directive.args":   dir.VisibleArgs,
		"__Field.args":       field.VisibleArgs,
		"__Type.inputFields": input.VisibleInputFields,
	}
	for name, visible := range sources {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, []string{"current"}, inputValueNames(visible(false)))

			all := visible(true)
			require.Equal(t, []string{"old", "current"}, inputValueNames(all))
			require.True(t, all[0].IsDeprecated())
			require.Equal(t, "gone", *all[0].DeprecationReason())
			require.False(t, all[1].IsDeprecated())
		})
	}
}

func inputValueNames(values []InputValue) []string {
	names := make([]string, len(values))
	for i, v := range values {
		names[i] = v.Name
	}
	return names
}
