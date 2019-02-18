package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("config does not exist", func(t *testing.T) {
		_, err := LoadConfig("doesnotexist.yml")
		require.Error(t, err)
	})

	t.Run("malformed config", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/malformedconfig.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `asdf` into config.Config")
	})

	t.Run("unknown keys", func(t *testing.T) {
		_, err := LoadConfig("testdata/cfg/unknownkeys.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 2: field unknown not found in type config.Config")
	})
}

func TestLoadDefaultConfig(t *testing.T) {
	testDir, err := os.Getwd()
	require.NoError(t, err)
	var cfg *Config

	t.Run("will find closest match", func(t *testing.T) {
		err = os.Chdir(filepath.Join(testDir, "testdata", "cfg", "subdir"))
		require.NoError(t, err)

		cfg, err = LoadConfigFromDefaultLocations()
		require.NoError(t, err)
		require.Equal(t, StringList{"inner"}, cfg.SchemaFilename)
	})

	t.Run("will find config in parent dirs", func(t *testing.T) {
		err = os.Chdir(filepath.Join(testDir, "testdata", "cfg", "otherdir"))
		require.NoError(t, err)

		cfg, err = LoadConfigFromDefaultLocations()
		require.NoError(t, err)
		require.Equal(t, StringList{"outer"}, cfg.SchemaFilename)
	})

	t.Run("will return error if config doesn't exist", func(t *testing.T) {
		err = os.Chdir(testDir)
		require.NoError(t, err)

		cfg, err = LoadConfigFromDefaultLocations()
		require.True(t, os.IsNotExist(err))
	})
}

func TestReferencedPackages(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tm := TypeMap{
			"Foo": {Model: StringList{"github.com/test.Foo"}},
			"Bar": {Model: StringList{"github.com/test.Bar"}},
			"Baz": {Model: StringList{"github.com/otherpkg.Baz"}},
			"Map": {Model: StringList{"map[string]interface{}"}},
			"SkipResolver": {
				Fields: map[string]TypeMapField{
					"field": {Resolver: false},
				},
			},
		}

		pkgs := tm.ReferencedPackages()

		assert.Equal(t, []string{"github.com/test", "github.com/otherpkg"}, pkgs)
	})

}

func TestConfigCheck(t *testing.T) {
	t.Run("invalid config format due to conflicting package names", func(t *testing.T) {
		config, err := LoadConfig("testdata/cfg/conflictedPackages.yml")
		require.NoError(t, err)

		err = config.normalize()
		require.NoError(t, err)

		err = config.Check()
		require.EqualError(t, err, "filenames exec.go and models.go are in the same directory but have different package definitions")
	})
}
