package codegen

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

// A type alias and its target share a uniqueness key, so only one of them can
// spell the marshal and unmarshal helpers. Which one is picked must not depend
// on map iteration order, or regenerating the same schema produces a diff.
func TestAliasAndTargetReferencesAreChosenDeterministically(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/aliasbinding/gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.LoadSchema())
	require.NoError(t, cfg.Init())

	spellings := map[string]map[string]bool{}
	for range 100 {
		data, err := BuildData(cfg)
		require.NoError(t, err)
		for key, ref := range data.ReferencedTypes {
			if ref.Definition.Name != "Access" {
				continue
			}
			if spellings[key] == nil {
				spellings[key] = map[string]bool{}
			}
			spellings[key][types.TypeString(ref.GO, nil)] = true
		}
	}

	require.NotEmpty(t, spellings, "no reference to the Access enum was built")
	for key, seen := range spellings {
		require.Len(t, seen, 1, "%s was spelled differently across builds: %v", key, seen)
	}
}
