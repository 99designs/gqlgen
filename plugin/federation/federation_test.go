package federation

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
)

func TestWithEntities(t *testing.T) {
	f, cfg := load(t, "test_data/gqlgen.yml")

	require.Equal(t, []string{"ExternalExtension", "Hello", "World"}, cfg.Schema.Types["_Entity"].Types)

	require.Equal(t, "findExternalExtensionByUpc", cfg.Schema.Types["Entity"].Fields[0].Name)
	require.Equal(t, "findHelloByName", cfg.Schema.Types["Entity"].Fields[1].Name)
	require.Equal(t, "findWorldByFooAndBar", cfg.Schema.Types["Entity"].Fields[2].Name)

	require.NoError(t, f.MutateConfig(cfg))
}

func TestNoEntities(t *testing.T) {
	f, cfg := load(t, "test_data/nokey.yml")

	err := f.MutateConfig(cfg)
	require.NoError(t, err)
}

func load(t *testing.T, name string) (*federation, *config.Config) {
	t.Helper()

	cfg, err := config.LoadConfig(name)
	require.NoError(t, err)

	f := &federation{}
	cfg.Sources = append(cfg.Sources, f.InjectSourceEarly())
	require.NoError(t, cfg.LoadSchema())

	if src := f.InjectSourceLate(cfg.Schema); src != nil {
		cfg.Sources = append(cfg.Sources, src)
	}
	require.NoError(t, cfg.LoadSchema())

	require.NoError(t, cfg.Init())
	return f, cfg
}
