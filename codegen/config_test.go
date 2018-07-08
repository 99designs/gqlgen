package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("config does not exist", func(t *testing.T) {
		_, err := LoadConfig("doesnotexist.yml")
		require.EqualError(t, err, "unable to read config: open doesnotexist.yml: no such file or directory")
	})

	t.Run("malformed config", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/malformedconfig.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `asdf` into codegen.Config")
	})
}

func TestLoadDefaultConfig(t *testing.T) {
	testDir, err := os.Getwd()
	require.NoError(t, err)
	var cfg *Config

	t.Run("will find closest match", func(t *testing.T) {
		err = os.Chdir(filepath.Join(testDir, "testdata", "cfg", "subdir"))
		require.NoError(t, err)

		cfg, err = LoadDefaultConfig()
		require.NoError(t, err)
		require.Equal(t, cfg.SchemaFilename, "inner")
	})

	t.Run("will find config in parent dirs", func(t *testing.T) {
		err = os.Chdir(filepath.Join(testDir, "testdata", "cfg", "otherdir"))
		require.NoError(t, err)

		cfg, err = LoadDefaultConfig()
		require.NoError(t, err)
		require.Equal(t, cfg.SchemaFilename, "outer")
	})

	t.Run("will fallback to defaults", func(t *testing.T) {
		err = os.Chdir(testDir)
		require.NoError(t, err)

		cfg, err = LoadDefaultConfig()
		require.NoError(t, err)
		require.Equal(t, cfg.SchemaFilename, "schema.graphql")
	})
}
