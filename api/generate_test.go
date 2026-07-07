package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	// follow-schema layout emits one exec file per schema file instead of a single one.
	// Deliberately narrow: a *.resolvers.go glob would delete committed fixture files.
	matches, _ := filepath.Glob(filepath.Join(workDir, "graph", "*.generated.go"))
	for _, f := range matches {
		_ = os.Remove(f)
	}
}

func TestGenerate(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	tests := []struct {
		name    string
		workDir string
		// wantExec maps a snippet to the exact number of times it must appear in the
		// generated exec file. Generate compiles its own output (see
		// TestGenerateValidatesOutput), so these only pin down details that would
		// compile either way — in particular that a type registers its batch parents
		// exactly once, not once per enclosing list.
		wantExec map[string]int
		// execFile overrides which generated file wantExec is checked against, relative
		// to workDir. Needed for follow-schema, which has no single exec filename.
		execFile string
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
			// resolver.batch applies to every eligible type, so both schema types must
			// still register parents. Guards against the per-type filter over-filtering.
			name:    "batchresolver_global",
			workDir: filepath.Join(wd, "testdata", "batchresolver_global"),
			wantExec: map[string]int{
				// User.posts is the schema's only batch resolver, so User is the only
				// type that needs parents registered.
				`graphql.WithBatchParents(ctx, "User", v, nil)`: 1,
				// Post is marshaled as a list but resolves nothing through a batch
				// resolver, so it must not register.
				`graphql.WithBatchParents(ctx, "Post", v, nil)`: 0,
				// TypeSupportsBatchResolver rejects any type prefixed "__", so even a
				// global resolver.batch leaves introspection unbatched. Registering
				// parents for it was always dead work.
				`graphql.WithBatchParentValues(ctx, "__Directive", v)`: 0,
			},
		},
		{
			// omit_slice_element_pointers makes User.posts a []model.Post, so the
			// batch parents for Post must be normalised to []*model.Post. Without
			// that, the generated type assertion fails at runtime and every parent
			// silently falls back to a single-parent resolver call.
			name:    "batchresolver_value_slices",
			workDir: filepath.Join(wd, "testdata", "batchresolver_valueslices"),
			wantExec: map[string]int{
				`graphql.WithBatchParentValues(ctx, "Post", v)`: 1,
				// Only types that actually have a batch field register parents. The
				// introspection types never do unless resolver.batch is set globally,
				// so they must not pay for Post's batch resolver.
				`graphql.WithBatchParentValues(ctx, "__Directive", v)`: 0,
				`graphql.WithBatchParentValues(ctx, "__Type", v)`:      0,
			},
		},
		{
			// Cat is declared in a different schema file from the Animal interface, so
			// under follow-schema the Data instance rendering Animal's marshaler holds
			// only Dog. Both implementors must still be collected: dropping Cat is
			// silent, turning its batch resolver back into an N+1.
			name:     "batchresolver_follow_schema",
			workDir:  filepath.Join(wd, "testdata", "batchresolver_followschema"),
			execFile: filepath.Join("graph", "a_animal.generated.go"),
			wantExec: map[string]int{
				`graphql.WithBatchParents(ctx, "Dog", batchItemsDog, batchIdxMapDog)`: 1,
				`graphql.WithBatchParents(ctx, "Cat", batchItemsCat, batchIdxMapCat)`: 1,
			},
		},
		{
			// Only the innermost list registers batch parents. Before the guard on
			// nested lists this failed to compile for the interface case, and
			// registered a [][]Thing under "Thing" for the object case.
			name:    "batchresolver_nested_list",
			workDir: filepath.Join(wd, "testdata", "batchresolver_nestedlist"),
			wantExec: map[string]int{
				`graphql.WithBatchParents(ctx, "Thing", v, nil)`:                      1,
				`graphql.WithBatchParents(ctx, "Dog", batchItemsDog, batchIdxMapDog)`: 1,
			},
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

			if tt.wantExec == nil {
				return
			}
			execPath := cfg.Exec.Filename
			if tt.execFile != "" {
				execPath = tt.execFile
			}
			exec, err := os.ReadFile(execPath)
			require.NoError(t, err, "failed to read generated exec file")
			for snippet, want := range tt.wantExec {
				require.Equal(t, want, strings.Count(string(exec), snippet),
					"unexpected number of occurrences of %s", snippet)
			}
		})
	}
}

// TestGenerateValidatesOutput pins the fact that Generate compiles what it generates.
//
// The fixture's package contains a file that deliberately does not compile. Validation
// once built an import-path pattern, whose "..." wildcard skips testdata directories, so
// it matched no packages and reported success without compiling anything — leaving every
// fixture in this directory unchecked. If that regresses, this test fails.
func TestGenerateValidatesOutput(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	workDir := filepath.Join(wd, "testdata", "validation_catches_errors")
	t.Cleanup(func() {
		cleanup(workDir)
		t.Chdir(wd)
	})
	t.Chdir(workDir)

	cfg, err := config.LoadConfigFromDefaultLocations()
	require.NoError(t, err, "failed to load config")

	err = Generate(cfg)
	require.Error(t, err, "generation should fail when the generated package does not compile")
	require.Contains(t, err.Error(), "validation failed")
	require.Contains(t, err.Error(), "ThisTypeDoesNotExist")
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

func TestBuildPattern(t *testing.T) {
	cases := map[string]struct {
		dir  string
		want string
	}{
		// The "./" prefix is what makes go treat this as a file path. Without it,
		// "graph/..." is an import path, whose wildcard skips testdata directories.
		"relative dir":   {dir: "graph", want: "./graph/..."},
		"nested dir":     {dir: filepath.Join("internal", "graph"), want: "./internal/graph/..."},
		"current dir":    {dir: ".", want: "./..."},
		"empty dir":      {dir: "", want: "./..."},
		"already dotted": {dir: "./graph", want: "./graph/..."},
		"parent dir":     {dir: filepath.Join("..", "other"), want: "../other/..."},
		"absolute dir":   {dir: "/tmp/graph", want: "/tmp/graph/..."},
		// PackageConfig.Dir runs through filepath.Abs and ToSlash, so on Windows it
		// arrives rooted at a drive letter. Prefixing "./" there produces "./D:/..."
		// which the go command rejects, so these must hold on every OS, not just
		// where filepath.IsAbs happens to understand the shape.
		"windows drive":       {dir: "D:/a/gqlgen/graph", want: "D:/a/gqlgen/graph/..."},
		"windows drive lower": {dir: "c:/proj/graph", want: "c:/proj/graph/..."},
		"windows unc":         {dir: "//server/share/graph", want: "//server/share/graph/..."},
		"colon not a drive":   {dir: "we:rd/graph", want: "./we:rd/graph/..."},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, buildPattern(tc.dir))
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
	preExisting := []byte("// the previously-generated file — must survive a failed regen\npackage graph\n")
	require.NoError(t, os.MkdirAll(filepath.Dir(execPath), 0o755))
	require.NoError(t, os.WriteFile(execPath, preExisting, 0o644))

	// Run Generate with an erroring schema mutator: it fails AFTER schema load but BEFORE the
	// exec file is rewritten — the exact window where the old upfront-unlink left the file absent.
	err = Generate(cfg, AddPlugin(&testSchemaMutator{name: "error-mutator", shouldError: true}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "deliberate schema mutation error")

	// The pre-existing file must be INTACT — not absent, not truncated, not empty.
	got, readErr := os.ReadFile(execPath)
	require.NoError(t, readErr, "exec output must be PRESENT after a failed regen (was deleted under the old unlink-first behavior)")
	require.Equal(t, preExisting, got, "exec output must be UNCHANGED after a failed regen")
}
