package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/99designs/gqlgen/plugin/resolvergen"
)

type testPlugin struct{}

// Name returns the plugin name
func (t *testPlugin) Name() string {
	return "modelgen"
}

// MutateConfig mutates the configuration
func (t *testPlugin) MutateConfig(_ *config.Config) error {
	return nil
}

func mustFederationPlugin(t *testing.T) plugin.Plugin {
	p, err := federation.New(1, &config.Config{
		Federation: config.PackageConfig{},
	})
	if err != nil {
		require.Fail(t, "failed to create federation plugin")
	}
	return p
}

func TestReplacePlugin(t *testing.T) {
	t.Run("replace plugin if exists", func(t *testing.T) {
		pg := []plugin.Plugin{
			mustFederationPlugin(t),
			modelgen.New(),
			resolvergen.New(),
		}

		expectedPlugin := &testPlugin{}
		ReplacePlugin(expectedPlugin)(config.DefaultConfig(), &pg)

		require.EqualValues(t, mustFederationPlugin(t), pg[0])
		require.EqualValues(t, expectedPlugin, pg[1])
		require.EqualValues(t, resolvergen.New(), pg[2])
	})

	t.Run("add plugin if doesn't exist", func(t *testing.T) {
		pg := []plugin.Plugin{
			mustFederationPlugin(t),
			resolvergen.New(),
		}

		expectedPlugin := &testPlugin{}
		ReplacePlugin(expectedPlugin)(config.DefaultConfig(), &pg)

		require.EqualValues(t, mustFederationPlugin(t), pg[0])
		require.EqualValues(t, resolvergen.New(), pg[1])
		require.EqualValues(t, expectedPlugin, pg[2])
	})

	t.Run("do nothing if plugins is nil", func(t *testing.T) {
		ReplacePlugin(&testPlugin{})(config.DefaultConfig(), nil)
	})
}

func TestPrependPlugin(t *testing.T) {
	modelgenPlugin := modelgen.New()
	pg := []plugin.Plugin{
		modelgenPlugin,
	}

	expectedPlugin := &testPlugin{}
	PrependPlugin(expectedPlugin)(config.DefaultConfig(), &pg)

	require.EqualValues(t, expectedPlugin, pg[0])
	require.EqualValues(t, modelgenPlugin, pg[1])
}
