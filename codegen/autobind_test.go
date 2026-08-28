package codegen

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

func TestFindBindTarget_Autobind(t *testing.T) {
	input := `
package test

type Model struct {
	Name string
}

func (m Model) GetName() string {
	return m.Name
}

func (m Model) HasName() bool {
	return true
}
`
	scope, err := parseScope(input, "test")
	require.NoError(t, err)

	model := scope.Lookup("Model").Type().(*types.Named)

	tests := []struct {
		Name                string
		Model               types.Type
		Field               string
		AutoBindGetterHaser bool
		Expected            string // Expected method/field name
	}{
		{
			Name:                "Autobind enabled, should find GetName",
			Model:               model,
			Field:               "name",
			AutoBindGetterHaser: true,
			Expected:            "GetName",
		},
		{
			Name:                "Autobind disabled, should find Name field",
			Model:               model,
			Field:               "name",
			AutoBindGetterHaser: false,
			Expected:            "Name",
		},
		{
			Name: "Binding on an alias of the Model",
			Model: types.NewAlias(
				types.NewTypeName(0, nil, "AliasModel", nil),
				model,
			),
			Field:               "name",
			AutoBindGetterHaser: false,
			Expected:            "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			b := builder{Config: &config.Config{}}
			target, err := b.findBindTarget(tt.Model, tt.Field, tt.AutoBindGetterHaser)
			require.NoError(t, err)
			require.NotNil(t, target)
			require.Equal(t, tt.Expected, target.Name())
		})
	}
}
