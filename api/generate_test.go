package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
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
}
