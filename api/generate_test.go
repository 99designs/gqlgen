package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
)

func cleanup(workDir string) {
	_ = os.Remove(filepath.Join(workDir, "server.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "generated.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "resolver.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "federation.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "schema.resolvers.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "model", "models_gen.go"))
}

func TestGenerate(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	tests := []struct {
		name    string
		workDir string
	}{
		{
			name:    "default",
			workDir: filepath.Join(wd, "testdata", "default"),
		},
		{
			name:    "federation2",
			workDir: filepath.Join(wd, "testdata", "federation2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cleanup(tt.workDir)
				_ = os.Chdir(wd)
			})
			err = os.Chdir(tt.workDir)
			require.NoError(t, err)
			cfg, err := config.LoadConfigFromDefaultLocations()
			require.NoError(t, err, "failed to load config")
			err = Generate(cfg)
			require.NoError(t, err, "failed to generate code")
		})
	}

	t.Run("with InjectOperationSourcesEarly success", func(t *testing.T) {
		workDir := filepath.Join(wd, "testdata", "federation2")
		t.Cleanup(func() {
			cleanup(workDir)
			_ = os.Chdir(wd)
		})
		err = os.Chdir(workDir)
		require.NoError(t, err)
		cfg, err := config.LoadConfigFromDefaultLocations()
		require.NoError(t, err, "failed to load config")

		err = Generate(cfg, AddPlugin(&injectOperationSourcesEarlyPlugin{
			withError: false,
		}))
		require.NoError(t, err, "failed to generate code")
		require.Equal(t, []string{"query { todos {id} }"}, cfg.OperationSources)
	})

	t.Run("with InjectOperationSourcesEarly error", func(t *testing.T) {
		workDir := filepath.Join(wd, "testdata", "federation2")
		t.Cleanup(func() {
			cleanup(workDir)
			_ = os.Chdir(wd)
		})
		err = os.Chdir(workDir)
		require.NoError(t, err)
		cfg, err := config.LoadConfigFromDefaultLocations()
		require.NoError(t, err, "failed to load config")

		err = Generate(cfg, AddPlugin(&injectOperationSourcesEarlyPlugin{
			withError: true,
		}))
		require.Error(t, err, "failed to generate code")
	})

	t.Run("with InjectOperationSourcesLate success", func(t *testing.T) {
		workDir := filepath.Join(wd, "testdata", "federation2")
		t.Cleanup(func() {
			cleanup(workDir)
			_ = os.Chdir(wd)
		})
		err = os.Chdir(workDir)
		require.NoError(t, err)
		cfg, err := config.LoadConfigFromDefaultLocations()
		require.NoError(t, err, "failed to load config")

		err = Generate(cfg, AddPlugin(&injectOperationSourcesLatePlugin{
			withError: false,
		}))
		require.NoError(t, err, "failed to generate code")
		require.Equal(t, []string{"query { todos {id} }"}, cfg.OperationSources)
	})

	t.Run("with InjectOperationSourcesLate error", func(t *testing.T) {
		workDir := filepath.Join(wd, "testdata", "federation2")
		t.Cleanup(func() {
			cleanup(workDir)
			_ = os.Chdir(wd)
		})
		err = os.Chdir(workDir)
		require.NoError(t, err)
		cfg, err := config.LoadConfigFromDefaultLocations()
		require.NoError(t, err, "failed to load config")

		err = Generate(cfg, AddPlugin(&injectOperationSourcesLatePlugin{
			withError: true,
		}))
		require.Error(t, err, "failed to generate code")
	})
}

type injectOperationSourcesEarlyPlugin struct {
	withError bool
}

func (p *injectOperationSourcesEarlyPlugin) Name() string {
	return "My_InjectOperationSourcesEarly"
}

func (p *injectOperationSourcesEarlyPlugin) InjectOperationSourcesEarly() ([]string, error) {
	if p.withError {
		return nil, errors.New("InjectOperationSourcesEarlyPlugin error")
	}

	return []string{"query { todos {id} }"}, nil
}

type injectOperationSourcesLatePlugin struct {
	withError bool
}

func (p *injectOperationSourcesLatePlugin) Name() string {
	return "My_InjectOperationSourcesLate"
}

func (p *injectOperationSourcesLatePlugin) InjectOperationSourcesLate(schema *ast.Schema) ([]string, error) {
	if p.withError {
		return nil, errors.New("injectOperationSourcesLatePlugin error")
	}

	return []string{"query { todos {id} }"}, nil
}
