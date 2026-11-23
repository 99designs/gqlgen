package codegen

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

// TestFindBindTarget_StructTagOptions pins how a struct tag value is compared
// to a GraphQL field name. encoding/json allows options after the name, and
// gqlgen's own modelgen emits `omitzero` (and `omitempty,omitzero`) on nullable
// fields, so tag binding must read the name and ignore the options.
//
// Every Go field name is deliberately unrelated to its tag name, so the
// name-based fallback in findBindFieldTarget cannot mask a failed tag match.
func TestFindBindTarget_StructTagOptions(t *testing.T) {
	input := `
package test

type Model struct {
	Alpha string ` + "`" + `json:"plain"` + "`" + `
	Bravo string ` + "`" + `json:"omit,omitempty"` + "`" + `
	Delta string ` + "`" + `json:"omit_zero,omitzero"` + "`" + `
	Echo  string ` + "`" + `json:"both,omitempty,omitzero"` + "`" + `
	Gamma string ` + "`" + `json:"string_opt,string"` + "`" + `
}
`
	scope, err := parseScope(input, "test")
	require.NoError(t, err)

	model := scope.Lookup("Model").Type().(*types.Named)

	tests := []struct{ field, want string }{
		{"plain", "Alpha"},
		{"omit", "Bravo"},
		{"omit_zero", "Delta"},
		{"both", "Echo"},
		{"string_opt", "Gamma"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			b := builder{Config: &config.Config{StructTag: "json"}}
			target, err := b.findBindTarget(model, tt.field, false)
			require.NoError(t, err)
			require.NotNil(t, target, "tag %q should bind to %s", tt.field, tt.want)
			require.Equal(t, tt.want, target.Name())
		})
	}
}

// TestFindBindTarget_StructTagAmbiguousWithOptions pins that duplicate tag
// names are still reported as ambiguous when the tags carry extra options.
func TestFindBindTarget_StructTagAmbiguousWithOptions(t *testing.T) {
	input := `
package test

type Model struct {
	A string ` + "`" + `json:"dup,omitzero"` + "`" + `
	B string ` + "`" + `json:"dup,omitzero"` + "`" + `
}
`
	scope, err := parseScope(input, "test")
	require.NoError(t, err)

	model := scope.Lookup("Model").Type().(*types.Named)

	b := builder{Config: &config.Config{StructTag: "json"}}
	target, err := b.findBindTarget(model, "dup", false)
	require.Nil(t, target)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ambiguous")
}
