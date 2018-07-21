package codegen

import (
	"go/build"
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
		_, err := LoadConfig("tests/cfg/malformedconfig.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `asdf` into codegen.Config")
	})

	t.Run("unknown keys", func(t *testing.T) {
		_, err := LoadConfig("tests/cfg/unknownkeys.yml")
		require.EqualError(t, err, "unable to parse config: yaml: unmarshal errors:\n  line 2: field unknown not found in type codegen.Config")
	})
}

func TestLoadDefaultConfig(t *testing.T) {
	testDir, err := os.Getwd()
	require.NoError(t, err)
	var cfg *Config

	t.Run("will find closest match", func(t *testing.T) {
		err = os.Chdir(filepath.Join(testDir, "tests", "cfg", "subdir"))
		require.NoError(t, err)

		cfg, err = LoadDefaultConfig()
		require.NoError(t, err)
		require.Equal(t, cfg.SchemaFilename, "inner")
	})

	t.Run("will find config in parent dirs", func(t *testing.T) {
		err = os.Chdir(filepath.Join(testDir, "tests", "cfg", "otherdir"))
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

func Test_fullPackageName(t *testing.T) {
	origBuildContext := build.Default
	defer func() { build.Default = origBuildContext }()

	t.Run("gopath longer than package name", func(t *testing.T) {
		p := PackageConfig{Filename: "/b/src/y/foo/bar/baz.go"}
		build.Default.GOPATH = "/a/src/xxxxxxxxxxxxxxxxxxxxxxxx:/b/src/y"
		var got string
		ok := assert.NotPanics(t, func() { got = p.ImportPath() })
		if ok {
			assert.Equal(t, "/b/src/y/foo/bar", got)
		}
	})
	t.Run("stop searching on first hit", func(t *testing.T) {
		p := PackageConfig{Filename: "/a/src/x/foo/bar/baz.go"}
		build.Default.GOPATH = "/a/src/x:/b/src/y"
		var got string
		ok := assert.NotPanics(t, func() { got = p.ImportPath() })
		if ok {
			assert.Equal(t, "/a/src/x/foo/bar", got)
		}
	})
}
