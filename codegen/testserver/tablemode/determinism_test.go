//go:build !race

// See compile_test.go for why the tests that run the generator do not build with -race.

package tablemode_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
)

// TestGenerationIsDeterministic generates the executor of a schema twice in table mode
// and requires the same files: what table mode derives from maps, such as the names of
// the tables and the order in which imports get their aliases, must not follow the
// order of a map, which differs between the two runs.
func TestGenerationIsDeterministic(t *testing.T) {
	requireTableMode(t)
	for _, tc := range []struct{ name, config string }{
		{"single file", "table/gqlgen.yml"},
		{"follow-schema", "../followschema/gqlgen.yml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath, err := filepath.Abs(tc.config)
			require.NoError(t, err)
			first := generate(t, configPath)
			second := generate(t, configPath)
			require.Equal(t, first, second)
		})
	}
}

// generate generates the executor of the config, as generateDir does, and returns the
// generated files by name.
func generate(t *testing.T, configPath string) map[string]string {
	t.Helper()
	dir := generateDir(t, configPath)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	files := make(map[string]string, len(entries))
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		files[e.Name()] = string(b)
	}
	require.NotEmpty(t, files)
	return files
}

// generateDir generates the executor of the config, in the exec mode that the test binary
// is built for, into a temporary directory next to the config, where it has an import
// path, and returns the directory. The resolvers of the test server are not generated
// again.
func generateDir(t *testing.T, configPath string) string {
	t.Helper()
	dir, err := generateWith(t, configPath)
	require.NoError(t, err)
	return dir
}

// generateWith is generateDir with the options of api.Generate, such as plugins, and
// returns the error of the generation.
func generateWith(t *testing.T, configPath string, options ...api.Option) (string, error) {
	t.Helper()
	// The paths of a config are relative to its directory, where gqlgen runs.
	t.Chdir(filepath.Dir(configPath))
	cfg, err := config.LoadConfig(filepath.Base(configPath))
	require.NoError(t, err)
	// Go leaves directories whose name starts with _ out of ./... patterns. The
	// directory is inside the module, so that the generated package has an import path.
	//nolint:usetesting // see above
	dir, err := os.MkdirTemp(filepath.Dir(configPath), "_generated-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	if cfg.Exec.Layout == config.ExecLayoutFollowSchema {
		cfg.Exec.DirName = dir
	} else {
		cfg.Exec.Filename = filepath.Join(dir, "generated.go")
	}
	// The plugin that generates the models also declares the directives of gqlgen in
	// the schema, so it runs, into the temporary directory and the executor's package
	// rather than one named after the directory.
	cfg.Model.Filename = filepath.Join(dir, "models-gen.go")
	if cfg.Model.Package == "" {
		cfg.Model.Package = cfg.Exec.Package
	}
	if cfg.Federation.IsDefined() {
		cfg.Federation.Filename = filepath.Join(dir, "federation.go")
	}
	cfg.Resolver = config.ResolverConfig{}
	cfg.SkipValidation = true
	cfg.SkipModTidy = true
	return dir, api.Generate(cfg, options...)
}
