package api

import (
	"errors"
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
		{
			name:    "worker_limit",
			workDir: filepath.Join(wd, "testdata", "workerlimit"),
		},
		{
			name:    "batchresolver_global",
			workDir: filepath.Join(wd, "testdata", "batchresolver_global"),
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

// TestPerformanceOptionsWithAutobind tests that the three working performance
// optimization options (fast_validation, skip_import_grouping, use_buffer_pooling)
// work correctly with autobind and @goModel type mappings.
//
// This test validates that enabling these options doesn't cause:
// 1. Import cycles due to incorrect type detection
// 2. Missing or incorrect type mappings
// 3. Code generation failures
//
// The test scenario mirrors a common production pattern:
// - external package has LocationInfo type (mapped via @goModel)
// - model package (autobind) has Connection type referencing LocationInfo
// - external package imports model package (creating potential cycle)
func TestPerformanceOptionsWithAutobind(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	workDir := filepath.Join(wd, "testdata", "perf_options")
	t.Cleanup(func() {
		cleanup(workDir)
		t.Chdir(wd)
	})

	t.Chdir(workDir)

	cfg, err := config.LoadConfigFromDefaultLocations()
	require.NoError(t, err, "failed to load config")

	// Verify all three performance options are enabled
	require.True(t, cfg.GetFastValidation(), "fast_validation should be enabled")
	require.True(t, cfg.GetSkipImportGrouping(), "skip_import_grouping should be enabled")
	require.True(t, cfg.GetUseBufferPooling(), "use_buffer_pooling should be enabled")

	// Generate code with all optimization options enabled
	err = Generate(cfg)
	require.NoError(t, err, "generation failed with performance options enabled")

	// Read the generated models file to verify correctness
	modelsPath := filepath.Join(workDir, "graph", "model", "models_gen.go")
	content, err := os.ReadFile(modelsPath)
	require.NoError(t, err, "failed to read generated models file")

	contentStr := string(content)

	// The generated file should NOT import the external package directly.
	// If it does, it means optimization options broke autobind detection.
	require.NotContains(
		t,
		contentStr,
		`"github.com/99designs/gqlgen/api/testdata/perf_options/external"`,
		"models_gen.go should not import external package - this would cause an import cycle",
	)

	// Verify that Connection and Session types are NOT regenerated
	// (they should be used from the autobind package)
	require.NotContains(t, contentStr, "type Connection struct",
		"Connection should not be regenerated - it's in the autobind package")
	require.NotContains(t, contentStr, "type Session struct",
		"Session should not be regenerated - it's in the autobind package")

	// Verify the generated code includes expected content
	require.Contains(t, contentStr, "package model",
		"generated file should be in model package")
}

// TestPerformanceOptionsIndividually tests each performance option in isolation
// to ensure they don't interfere with correct code generation.
func TestPerformanceOptionsIndividually(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name               string
		fastValidation     bool
		skipImportGrouping bool
		useBufferPooling   bool
	}{
		{
			name:               "fast_validation_only",
			fastValidation:     true,
			skipImportGrouping: false,
			useBufferPooling:   false,
		},
		{
			name:               "skip_import_grouping_only",
			fastValidation:     false,
			skipImportGrouping: true,
			useBufferPooling:   false,
		},
		{
			name:               "use_buffer_pooling_only",
			fastValidation:     false,
			skipImportGrouping: false,
			useBufferPooling:   true,
		},
		{
			name:               "all_options_enabled",
			fastValidation:     true,
			skipImportGrouping: true,
			useBufferPooling:   true,
		},
		{
			name:               "no_options_enabled",
			fastValidation:     false,
			skipImportGrouping: false,
			useBufferPooling:   false,
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
			require.NoError(t, err, "failed to load config")

			// Override performance options for this test
			cfg.FastValidation = &tt.fastValidation
			cfg.SkipImportGrouping = &tt.skipImportGrouping
			cfg.UseBufferPooling = &tt.useBufferPooling

			// Generate code
			err = Generate(cfg)
			require.NoError(t, err, "generation failed with %s", tt.name)

			// Verify generated file exists and is valid
			modelsPath := filepath.Join(workDir, "graph", "model", "models_gen.go")
			content, err := os.ReadFile(modelsPath)
			require.NoError(t, err, "failed to read generated models file")
			require.Contains(t, string(content), "package model")
		})
	}
}

// TestGenerateAtomicWritePreservesOutputOnFailure reproduces issue #2345/#3505: when generation
// FAILS mid-run (after the output file would have been deleted, before it's rewritten), the
// PRE-EXISTING generated file must SURVIVE intact — not be left absent. Before the atomic-write
// fix, api/generate.go unlinked the exec + model outputs at the very start of Generate, so a
// failure anywhere in the long schema-load/plugin/render chain left the file ABSENT (invisible
// to `go build` until the next regen, and the root of the recovery-desyncs-sibling-golden class).
// Now the write is atomic (write-to-temp + os.Rename), and the upfront unlink is gone, so a
// mid-generation failure leaves the prior file untouched.
func TestGenerateAtomicWritePreservesOutputOnFailure(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	workDir := filepath.Join(wd, "testdata", "default")
	t.Cleanup(func() {
		cleanup(workDir)
		t.Chdir(wd)
	})
	t.Chdir(workDir)

	cfg, err := config.LoadConfigFromDefaultLocations()
	require.NoError(t, err)

	// Pre-create the exec output with known content — the "previously generated" file that must
	// survive a failed regen.
	execPath := cfg.Exec.Filename
	preExisting := []byte(
		"// the previously-generated file — must survive a failed regen\npackage graph\n",
	)
	require.NoError(t, os.MkdirAll(filepath.Dir(execPath), 0o755))
	require.NoError(t, os.WriteFile(execPath, preExisting, 0o644))

	// Run Generate with an erroring schema mutator: it fails AFTER schema load but BEFORE the
	// exec file is rewritten — the exact window where the old upfront-unlink left the file absent.
	err = Generate(cfg, AddPlugin(&testSchemaMutator{name: "error-mutator", shouldError: true}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "deliberate schema mutation error")

	// The pre-existing file must be INTACT — not absent, not truncated, not empty.
	got, readErr := os.ReadFile(execPath)
	require.NoError(
		t,
		readErr,
		"exec output must be PRESENT after a failed regen (was deleted under the old unlink-first behavior)",
	)
	require.Equal(t, preExisting, got, "exec output must be UNCHANGED after a failed regen")
}
