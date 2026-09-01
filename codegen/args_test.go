package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

// bindArgs rebinds an argument to the type of the matching parameter of the Go
// method the field is bound to. The reference built from the default binding is
// superseded by that one, so it must not be left registered: nothing calls the
// unmarshaler that would be emitted for it.
func TestBindArgsDropsSupersededReference(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/argbinding/gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.LoadSchema())
	require.NoError(t, cfg.Init())

	data, err := BuildData(cfg)
	require.NoError(t, err)

	var got []string
	for _, ref := range data.ReferencedTypes {
		if ref.Definition.Name == "Int" {
			got = append(got, ref.GO.String())
		}
	}
	require.ElementsMatch(t, []string{"int64"}, got)
}
