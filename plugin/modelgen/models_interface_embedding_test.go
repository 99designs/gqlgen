package modelgen

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

func TestModelGenerationNoDirective(t *testing.T) {
	t.Run("generated code does not contains Base structs", func(t *testing.T) {
		generated := setupTestGeneration(t,
			"testdata/interface_embedding/gqlgen_no_directive_models.yml",
			"./out_no_directive_models/",
			"./out_no_directive_models/generated_no_directive_models.go",
		)

		require.NotContains(t, generated, "type Base")
	})

	t.Run("graph does not have embeddable interfaces", func(t *testing.T) {
		cfg, err := config.LoadConfig("testdata/interface_embedding/gqlgen_no_directive_models.yml")
		require.NoError(t, err)
		require.NoError(t, cfg.Init())

		embedder := newEmbeddedInterfaceGenerator(cfg, cfg.NewBinder(), nil, nil)
		require.NotEmpty(t, embedder.graph.parentInterfaces, "graph should contain all interfaces")

		for name := range embedder.graph.parentInterfaces {
			require.False(t, embedder.graph.isEmbeddable(name), "interface %s should not be embeddable", name)
		}
	})
}

func TestModelGenerationDirectiveEmbedding(t *testing.T) {
	testCases := []struct {
		name             string
		configPath       string
		outputDir        string
		generatedFile    string
		expectedBases    []string
		unexpectedBases  []string
		structChecks     []structCheck
		additionalChecks func(*testing.T, string)
	}{
		{
			name:          "single package base type embedding",
			configPath:    "testdata/interface_embedding/gqlgen_directive_embedding_models.yml",
			outputDir:     "./out_directive_embedding_models/",
			generatedFile: "./out_directive_embedding_models/generated_directive_embedding_models.go",
			expectedBases: []string{"BaseNode", "BaseElement"},
			structChecks: []structCheck{
				{"BaseElement", []string{"BaseNode", "Name"}, nil},
				{"BaseNode", []string{"ID"}, []string{"Name"}},
				{"Carbon", []string{"BaseElement"}, nil},
				{"Magnesium", []string{"BaseElement"}, nil},
				{"Potassium", []string{"BaseElement"}, nil},
			},
		},
		{
			name:            "binded package type embedding",
			configPath:      "testdata/interface_embedding/gqlgen_directive_binding_models.yml",
			outputDir:       "./out_directive_binding_models/",
			generatedFile:   "./out_directive_binding_models/generated_directive_binding_models.go",
			unexpectedBases: []string{"BaseElement", "BaseNode"},
			structChecks: []structCheck{
				{"Oxygen", []string{"out_directive_embedding_models.BaseElement", "Purity"}, nil},
				{"Molecule", []string{"out_directive_embedding_models.BaseNode"}, nil},
			},
			additionalChecks: func(t *testing.T, generated string) {
				require.Contains(t, generated, "github.com/99designs/gqlgen/plugin/modelgen/out_directive_embedding_models")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generated := setupTestGeneration(t, tc.configPath, tc.outputDir, tc.generatedFile)

			assertBaseStructPresence(t, generated, tc.expectedBases, tc.unexpectedBases)

			for _, check := range tc.structChecks {
				assertStructFields(t, generated, check.structName, check.mustContain, check.mustNotContain)
			}

			if tc.additionalChecks != nil {
				tc.additionalChecks(t, generated)
			}

			cfg, err := config.LoadConfig(tc.configPath)
			require.NoError(t, err)
			require.NoError(t, cfg.Init())

			embedder := newEmbeddedInterfaceGenerator(cfg, cfg.NewBinder(), nil, nil)
			specs, err := embedder.generateAllInterfaceBaseStructs()
			require.NoError(t, err)
			require.Len(t, specs, len(tc.expectedBases))
		})
	}
}

type structCheck struct {
	structName     string
	mustContain    []string
	mustNotContain []string
}

func TestModelGenerationDirectiveCovariantTypes(t *testing.T) {
	generated := setupTestGeneration(t,
		"testdata/interface_embedding/gqlgen_directive_covariant_types.yml",
		"./out_covariant_types/",
		"./out_covariant_types/generated_covariant_types.go",
	)

	t.Run("ProductNode uses ProductNodeData not embedded BaseNode", func(t *testing.T) {
		assertStructFields(t, generated, "ProductNode",
			[]string{"ID", "Type", "*ProductNodeData", "ProductTitle"},
			[]string{"BaseNode"},
		)
	})

	t.Run("ExtendedProductNode has covariant override for data", func(t *testing.T) {
		assertStructFields(t, generated, "ExtendedProductNode",
			[]string{"ID", "Type", "*ProductNodeData", "*ProductTags"},
			[]string{"BaseNode"},
		)
	})

	t.Run("ProductNodeData also handles covariant overrides", func(t *testing.T) {
		assertStructFields(t, generated, "ProductNodeData",
			[]string{"BaseNodeData", "ProductSpecificField"},
			nil,
		)
	})
}

func TestModelGenerationSkippedParents(t *testing.T) {
	testCases := []struct {
		name            string
		configPath      string
		outputDir       string
		generatedFile   string
		expectedBases   []string
		unexpectedBases []string
		structChecks    []structCheck
	}{
		{
			name:            "skipped parents with A->B->C hierarchy",
			configPath:      "testdata/interface_embedding/gqlgen_directive_skipped_parents.yml",
			outputDir:       "./out_directive_skipped_parents/",
			generatedFile:   "./out_directive_skipped_parents/generated_directive_skipped_parents.go",
			expectedBases:   []string{"BaseA", "BaseC"},
			unexpectedBases: []string{"BaseB"},
			structChecks: []structCheck{
				{"BaseC", []string{"BaseA", "FieldB"}, []string{"BaseB"}},
				{"ConcreteC", []string{"BaseC"}, nil},
			},
		},
		{
			name:            "partial directive with Node->Element->Metal hierarchy",
			configPath:      "testdata/interface_embedding/gqlgen_directive_partial.yml",
			outputDir:       "./out_directive_partial/",
			generatedFile:   "./out_directive_partial/generated_directive_partial.go",
			expectedBases:   []string{"BaseNode", "BaseMetal"},
			unexpectedBases: []string{"BaseElement"},
			structChecks: []structCheck{
				{"BaseMetal", []string{"BaseNode", "Name"}, []string{"BaseElement"}},
				{"Gold", []string{"BaseMetal"}, nil},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generated := setupTestGeneration(t, tc.configPath, tc.outputDir, tc.generatedFile)

			assertBaseStructPresence(t, generated, tc.expectedBases, tc.unexpectedBases)

			for _, check := range tc.structChecks {
				assertStructFields(t, generated, check.structName, check.mustContain, check.mustNotContain)
			}
		})
	}
}

func TestModelGenerationDirectiveDiamond(t *testing.T) {
	generated := setupTestGeneration(t,
		"testdata/interface_embedding/gqlgen_directive_diamond.yml",
		"./out_directive_diamond/",
		"./out_directive_diamond/generated_directive_diamond.go",
	)

	assertBaseStructPresence(t, generated,
		[]string{"BaseHasID", "BaseHasIdentifier", "BaseConflicting", "BaseNoConflict"},
		[]string{"BaseHasName", "BaseHasTitle"},
	)

	assertStructFields(t, generated, "BaseConflicting",
		[]string{"BaseHasID", "BaseHasIdentifier"},
		nil,
	)

	assertStructFields(t, generated, "BaseNoConflict",
		[]string{"Name", "Title"},
		nil,
	)
}

func setupTestGeneration(t *testing.T, configPath, outputDir, generatedFile string) string {
	t.Helper()
	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		FieldHook: DefaultFieldMutateHook,
	}
	require.NoError(t, p.MutateConfig(cfg))
	require.NoError(t, goBuild(t, outputDir))
	generated, err := os.ReadFile(generatedFile)
	require.NoError(t, err)
	return string(generated)
}

func assertBaseStructPresence(t *testing.T, generated string, expectedBases, unexpectedBases []string) {
	t.Helper()
	for _, base := range expectedBases {
		require.Contains(t, generated, "type "+base, "Expected Base struct %s to be present", base)
	}
	for _, base := range unexpectedBases {
		require.NotContains(t, generated, "type "+base, "Expected Base struct %s to be absent", base)
	}
}

func assertStructFields(t *testing.T, generated, structName string, mustContain, mustNotContain []string) {
	t.Helper()
	structStr := getStringInBetween(generated, "type "+structName+" struct {", "}")
	require.NotEmpty(t, structStr, "Struct %s should exist", structName)
	for _, field := range mustContain {
		require.Contains(t, structStr, field, "Struct %s should contain field %s", structName, field)
	}
	for _, field := range mustNotContain {
		require.NotContains(t, structStr, field, "Struct %s should not contain field %s", structName, field)
	}
}
