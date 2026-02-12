package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

func cleanup(workDir string) {
	_ = os.Remove(filepath.Join(workDir, "server.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "generated.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "split_runtime.generated.go"))
	for i := 0; i < 64; i++ {
		_ = os.Remove(filepath.Join(workDir, "graph", fmt.Sprintf("split_shard_import_%d.generated.go", i)))
	}
	_ = os.RemoveAll(filepath.Join(workDir, "graph", "internal", "gqlgenexec"))
	_ = os.Remove(filepath.Join(workDir, "graph", "resolver.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "federation.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "schema.resolvers.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "model", "models_gen.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "model", "resolver.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "model", "schema.resolvers.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "model", "generated.go"))
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
		{
			name:    "worker_limit",
			workDir: filepath.Join(wd, "testdata", "workerlimit"),
		},
		{
			name:    "split_packages",
			workDir: filepath.Join(wd, "testdata", "splitpackages"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cleanup(tt.workDir)
				t.Chdir(wd)
			})
			t.Chdir(tt.workDir)
			cfg, err := config.LoadConfigFromDefaultLocations()
			require.NoError(t, err, "failed to load config")
			err = Generate(cfg)
			require.NoError(t, err, "failed to generate code")
		})
	}
}

type testSchemaMutator struct {
	name        string
	shouldError bool
}

func (t *testSchemaMutator) Name() string {
	return t.name
}

func (t *testSchemaMutator) MutateSchema(schema *ast.Schema) error {
	if t.shouldError {
		return errors.New("deliberate schema mutation error")
	}
	schema.Types["TestType"] = &ast.Definition{
		Kind: ast.Object,
		Name: "TestType",
		Fields: ast.FieldList{
			{
				Name: "id",
				Type: ast.NamedType("ID", nil),
			},
		},
	}
	return nil
}

func TestGenerateWithSchemaMutator(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name        string
		mutator     *testSchemaMutator
		shouldError bool
	}{
		{
			name:        "successful schema mutation",
			mutator:     &testSchemaMutator{name: "test-mutator", shouldError: false},
			shouldError: false,
		},
		{
			name:        "failed schema mutation",
			mutator:     &testSchemaMutator{name: "error-mutator", shouldError: true},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := filepath.Join(wd, "testdata", "default")
			t.Cleanup(func() {
				cleanup(workDir)
				t.Chdir(wd)
			})

			t.Chdir(workDir)

			cfg, err := config.LoadConfigFromDefaultLocations()
			require.NoError(t, err)

			err = Generate(cfg, AddPlugin(tt.mutator))
			if tt.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "deliberate schema mutation error")
			} else {
				require.NoError(t, err)
				require.Contains(t, cfg.Schema.Types, "TestType")
				require.Equal(t, ast.Object, cfg.Schema.Types["TestType"].Kind)
			}
		})
	}
}
