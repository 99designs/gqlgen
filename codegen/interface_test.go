package codegen

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

// An interface that implements another interface, bound to a Go interface,
// must not get a pointer case in the parent's type switch: a pointer to an
// interface never holds the value, and ec._Captioned(ctx, sel, obj) with obj
// a *model.Captioned does not compile. Node bound to an empty interface is
// satisfied by every type, the pointer included, so the check alone does not
// rule it out (#4442).
func TestInterfaceImplementorBoundToGoInterfaceHasNoPointerCase(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/interfaceimplements/gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.LoadSchema())
	require.NoError(t, cfg.Init())

	data, err := BuildData(cfg)
	require.NoError(t, err)

	node := data.Interfaces["Node"]
	require.NotNil(t, node)

	var cases []string
	for _, impl := range node.Implementors {
		cases = append(cases, types.TypeString(impl.Type, (*types.Package).Name))
	}
	require.ElementsMatch(t, []string{"model.Post", "*model.Post", "model.Captioned"}, cases)

	for _, impl := range node.Implementors {
		if impl.Name == "Captioned" {
			require.False(t, impl.TakeRef, "an interface value is passed as it is")
		}
	}
}
