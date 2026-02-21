package codegen

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	internalcode "github.com/99designs/gqlgen/internal/code"
)

func TestLayoutGuardrailsUnchanged(t *testing.T) {
	singleDir := filepath.Join("testserver", "singlefile")
	followDir := filepath.Join("testserver", "followschema")

	singleExecFiles := guardrailExecGeneratedFiles(t, singleDir)
	require.Equal(t, []string{"generated.go"}, singleExecFiles)

	followExecFiles := guardrailExecGeneratedFiles(t, followDir)
	require.Contains(t, followExecFiles, "root_.generated.go")
	require.Contains(t, followExecFiles, "schema.generated.go")
	require.Greater(t, len(followExecFiles), 1)

	for _, rel := range singleExecFiles {
		require.NotContains(t, rel, "split_shard_import_")
		require.NotEqual(t, "split_runtime.generated.go", rel)
	}
	for _, rel := range followExecFiles {
		require.NotContains(t, rel, "split_shard_import_")
		require.NotEqual(t, "split_runtime.generated.go", rel)
	}

	for _, dir := range []string{singleDir, followDir} {
		_, err := os.Stat(filepath.Join(dir, "internal", "gqlgenexec"))
		require.True(t, os.IsNotExist(err), "non-split layout %s must not emit split shard directories", dir)
	}

	singleRoot := filepath.Join(singleDir, "generated.go")
	followRoot := filepath.Join(followDir, "root_.generated.go")

	guardrailRequireNewExecutableSignature(t, singleRoot)
	guardrailRequireNewExecutableSignature(t, followRoot)

	singleImports := guardrailImports(t, singleRoot)
	followImports := guardrailImports(t, followRoot)
	require.Contains(t, singleImports, "github.com/99designs/gqlgen/graphql")
	require.Contains(t, followImports, "github.com/99designs/gqlgen/graphql")
	require.NotContains(t, singleImports, "github.com/99designs/gqlgen/graphql/executor/shardruntime")
	require.NotContains(t, followImports, "github.com/99designs/gqlgen/graphql/executor/shardruntime")
}

func TestSplitPackagesFederationStillUnsupported(t *testing.T) {
	workDir := t.TempDir()
	configFile := filepath.Join(workDir, "gqlgen.yml")
	schemaFile := filepath.Join(workDir, "schema.graphqls")

	require.NoError(t, os.WriteFile(schemaFile, []byte("type Query { hello: String! }\n"), 0o644))
	require.NoError(t, os.WriteFile(configFile, []byte(`schema:
  - schema.graphqls
exec:
  layout: split-packages
  filename: graph/generated.go
  package: graph
federation:
  filename: graph/federation.go
  package: graph
`), 0o644))

	cfg, err := config.LoadConfig(configFile)
	require.NoError(t, err)
	err = cfg.Init()
	require.Error(t, err)
	require.ErrorContains(t, err, "federation is not supported with exec.layout=split-packages yet")
}

func TestSplitPackagesDeterminism(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	firstRun := generateSplitSnapshot(t)

	cleanupSplitGeneratedFiles(workDir)
	secondRun := generateSplitSnapshot(t)

	require.Equal(t, firstRun, secondRun)
}

func TestSplitOmitComplexityDoesNotReferenceComplexityRoot(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)
	configPath := filepath.Join(workDir, "gqlgen.yml")

	contents, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contents = append(bytes.TrimRight(contents, "\n"), []byte("\nomit_complexity: true\n")...)
	require.NoError(t, os.WriteFile(configPath, contents, 0o644))

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	generated, ok := snapshot[filepath.Join("graph", "generated.go")]
	require.True(t, ok)
	require.NotContains(t, string(generated), "ec.complexity.")
}

func guardrailExecGeneratedFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name == "generated.go" || strings.HasSuffix(name, ".generated.go") {
			files = append(files, name)
		}
	}

	sort.Strings(files)
	return files
}

func guardrailRequireNewExecutableSignature(t *testing.T, filePath string) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Name.Name != "NewExecutableSchema" {
			continue
		}

		require.Nil(t, fn.Recv)
		require.Len(t, fn.Type.Params.List, 1)
		require.NotEmpty(t, fn.Type.Params.List[0].Names)
		require.Equal(t, "cfg", fn.Type.Params.List[0].Names[0].Name)

		paramType, ok := fn.Type.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		require.Equal(t, "Config", paramType.Name)

		require.Len(t, fn.Type.Results.List, 1)
		resultType, ok := fn.Type.Results.List[0].Type.(*goast.SelectorExpr)
		require.True(t, ok)

		resultPkg, ok := resultType.X.(*goast.Ident)
		require.True(t, ok)
		require.Equal(t, "graphql", resultPkg.Name)
		require.Equal(t, "ExecutableSchema", resultType.Sel.Name)
		return
	}

	require.Failf(t, "missing NewExecutableSchema", "expected NewExecutableSchema in %s", filePath)
}

func guardrailImports(t *testing.T, filePath string) []string {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), filePath, nil, parser.ImportsOnly)
	require.NoError(t, err)

	imports := make([]string, 0, len(parsed.Imports))
	for _, imp := range parsed.Imports {
		importPath, unquoteErr := strconv.Unquote(imp.Path.Value)
		require.NoError(t, unquoteErr)
		imports = append(imports, importPath)
	}

	return imports
}

func TestSplitStaleFileCleanupDeterministic(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	baseline := generateSplitSnapshot(t)

	staleRootImport := filepath.Join(workDir, "graph", "split_shard_import_999.generated.go")
	staleSplitOwnedShardGenerated := filepath.Join(workDir, "graph", "internal", "gqlgenexec", "shards", "stale", "legacy.generated.go")
	staleUnownedShardGenerated := filepath.Join(workDir, "graph", "internal", "gqlgenexec", "shards", "stale", "foreign.generated.go")
	staleRegister := filepath.Join(workDir, "graph", "internal", "gqlgenexec", "shards", "obsolete", "register.generated.go")
	keptInsideShardScope := filepath.Join(workDir, "graph", "internal", "gqlgenexec", "shards", "stale", "keep.txt")
	unrelatedOutsideScope := filepath.Join(workDir, "graph", "unrelated.generated.go")

	require.NoError(t, os.MkdirAll(filepath.Dir(staleRootImport), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(staleSplitOwnedShardGenerated), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(staleRegister), 0o755))
	require.NoError(t, os.WriteFile(staleRootImport, []byte("package graph\n"), 0o644))
	require.NoError(t, os.WriteFile(staleSplitOwnedShardGenerated, []byte("package stale\nconst splitScope = \"scope\"\n"), 0o644))
	require.NoError(t, os.WriteFile(staleUnownedShardGenerated, []byte("package stale\n"), 0o644))
	require.NoError(t, os.WriteFile(staleRegister, []byte("package obsolete\n"), 0o644))
	require.NoError(t, os.WriteFile(keptInsideShardScope, []byte("keep me\n"), 0o644))
	require.NoError(t, os.WriteFile(unrelatedOutsideScope, []byte("package graph\n"), 0o644))

	afterCleanup := generateSplitSnapshot(t)
	secondRun := generateSplitSnapshot(t)

	require.Equal(t, baseline, afterCleanup)
	require.Equal(t, afterCleanup, secondRun)

	_, err := os.Stat(staleRootImport)
	require.True(t, os.IsNotExist(err), "expected stale split import to be removed")

	_, err = os.Stat(staleSplitOwnedShardGenerated)
	require.True(t, os.IsNotExist(err), "expected stale split shard file to be removed")

	staleUnownedContents, err := os.ReadFile(staleUnownedShardGenerated)
	require.NoError(t, err)
	require.Equal(t, "package stale\n", string(staleUnownedContents))

	_, err = os.Stat(staleRegister)
	require.True(t, os.IsNotExist(err), "expected stale split register file to be removed")

	_, err = os.Stat(filepath.Dir(staleRegister))
	require.True(t, os.IsNotExist(err), "expected empty stale shard directory to be pruned")

	keptContents, err := os.ReadFile(keptInsideShardScope)
	require.NoError(t, err)
	require.Equal(t, "keep me\n", string(keptContents))

	unrelatedContents, err := os.ReadFile(unrelatedOutsideScope)
	require.NoError(t, err)
	require.Equal(t, "package graph\n", string(unrelatedContents))

	t.Cleanup(func() {
		_ = os.Remove(unrelatedOutsideScope)
	})
}

func TestSplitStaleCleanupDeterministicAndScoped(t *testing.T) {
	TestSplitStaleFileCleanupDeterministic(t)
}

func TestListSplitShardGeneratedFilesSupportsCustomTemplate(t *testing.T) {
	root := t.TempDir()

	ownedCustom := filepath.Join(root, "custom", "legacy.go")
	ownedRegister := filepath.Join(root, "legacy", "register.generated.go")
	unownedCustom := filepath.Join(root, "custom", "foreign.go")
	nonMatching := filepath.Join(root, "custom", "legacy.txt")

	require.NoError(t, os.MkdirAll(filepath.Dir(ownedCustom), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(ownedRegister), 0o755))

	require.NoError(t, os.WriteFile(ownedCustom, []byte("package custom\nconst splitScope = \"scope\"\n"), 0o644))
	require.NoError(t, os.WriteFile(ownedRegister, []byte("package legacy\n"), 0o644))
	require.NoError(t, os.WriteFile(unownedCustom, []byte("package custom\n"), 0o644))
	require.NoError(t, os.WriteFile(nonMatching, []byte("package custom\nconst splitScope = \"scope\"\n"), 0o644))

	files, err := listSplitShardGeneratedFiles(root, "{name}.go")
	require.NoError(t, err)
	require.Equal(t, []string{ownedCustom, ownedRegister}, files)
}

func TestListSplitShardGeneratedFilesIncludesLegacyTemplateMatches(t *testing.T) {
	root := t.TempDir()

	legacyNamedOwned := filepath.Join(root, "legacy", "alpha.generated.go")
	currentNamedOwned := filepath.Join(root, "legacy", "beta.gql.go")
	registerFile := filepath.Join(root, "legacy", "register.generated.go")
	unownedLegacyNamed := filepath.Join(root, "legacy", "foreign.generated.go")

	require.NoError(t, os.MkdirAll(filepath.Dir(legacyNamedOwned), 0o755))
	require.NoError(t, os.WriteFile(legacyNamedOwned, []byte("package legacy\nconst splitScope = \"scope\"\n"), 0o644))
	require.NoError(t, os.WriteFile(currentNamedOwned, []byte("package legacy\nconst splitScope = \"scope\"\n"), 0o644))
	require.NoError(t, os.WriteFile(registerFile, []byte("package legacy\n"), 0o644))
	require.NoError(t, os.WriteFile(unownedLegacyNamed, []byte("package legacy\n"), 0o644))

	files, err := listSplitShardGeneratedFiles(root, "{name}.gql.go")
	require.NoError(t, err)
	require.Equal(t, []string{legacyNamedOwned, currentNamedOwned, registerFile}, files)
}

func TestSplitShortHashZeroPadsToFixedWidth(t *testing.T) {
	require.NotPanics(t, func() {
		_ = splitShortHash("x3130")
	})
	require.Equal(t, "0fae14", splitShortHash("x3130"))
	require.Len(t, splitShortHash("x3130"), 6)
}

func TestSplitPackagesShardNameCollision(t *testing.T) {
	newBuild := func(source string) *Data {
		return &Data{Config: &config.Config{Sources: []*ast.Source{{Name: source}}}}
	}

	firstUsed := map[string]string{}
	first := []string{
		splitShardName("a-b.generated.go", newBuild("graph/a-b.graphqls"), firstUsed),
		splitShardName("a_b.generated.go", newBuild("graph/a_b.graphqls"), firstUsed),
	}

	secondUsed := map[string]string{}
	second := []string{
		splitShardName("a-b.generated.go", newBuild("graph/a-b.graphqls"), secondUsed),
		splitShardName("a_b.generated.go", newBuild("graph/a_b.graphqls"), secondUsed),
	}

	require.Equal(t, first, second)
	require.Equal(t, []string{"a_b", "a_b_" + splitShortHash("a_b.generated.go")}, first)
}

func TestSplitPackagesShardNameKeywordSanitization(t *testing.T) {
	newBuild := func(source string) *Data {
		return &Data{Config: &config.Config{Sources: []*ast.Source{{Name: source}}}}
	}

	used := map[string]string{}
	shardName := splitShardName("type.generated.go", newBuild("graph/type.graphqls"), used)

	require.Equal(t, "s_type", shardName)
	require.False(t, token.Lookup(shardName).IsKeyword())
}

func TestSplitOwnershipPlannerDeterministic(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	data := buildSplitData(t)

	first, err := planSplitOwnership(data)
	require.NoError(t, err)

	shuffled := shuffledOwnershipData(data)
	second, err := planSplitOwnership(shuffled)
	require.NoError(t, err)

	require.Equal(t, first.FieldOwner, second.FieldOwner)
	require.Equal(t, first.ArgsOwner, second.ArgsOwner)
	require.Equal(t, first.FieldContextOwner, second.FieldContextOwner)
	require.Equal(t, first.ComplexityOwner, second.ComplexityOwner)

	require.Equal(t, sortedOwnershipKeys(first.FieldOwner), first.FieldOwnerKeys)
	require.Equal(t, sortedOwnershipKeys(first.ArgsOwner), first.ArgsOwnerKeys)
	require.Equal(t, sortedOwnershipKeys(first.FieldContextOwner), first.FieldContextOwnerKeys)
	require.Equal(t, sortedOwnershipKeys(first.ComplexityOwner), first.ComplexityOwnerKeys)

	require.NotEmpty(t, first.FieldOwner)
	require.NotEmpty(t, first.FieldContextOwner)
	require.NotEmpty(t, first.ComplexityOwner)
}

func TestSplitInputOwnerDeterministic(t *testing.T) {
	data := splitInputOwnerTestData()

	first, err := planSplitOwnership(data)
	require.NoError(t, err)

	shuffled := *data
	shuffled.Objects = Objects{data.Objects[1], data.Objects[0]}
	shuffled.Inputs = Objects{data.Inputs[2], data.Inputs[0], data.Inputs[1]}

	second, err := planSplitOwnership(&shuffled)
	require.NoError(t, err)

	require.Equal(t, first.InputOwner, second.InputOwner)
	require.Equal(t, "alpha", first.InputOwner["NestedInput"])
	require.Equal(t, "alpha", first.InputOwner["SharedInput"])
	require.Equal(t, "common", first.InputOwner["OrphanInput"])
	require.Equal(t, sortedOwnershipKeys(first.InputOwner), first.InputOwnerKeys)
}

func TestSplitCodecOwnerDeterministic(t *testing.T) {
	data := splitCodecOwnerTestData()

	first, err := planSplitOwnership(data)
	require.NoError(t, err)

	shuffled := *data
	shuffled.Objects = Objects{data.Objects[1], data.Objects[0]}

	second, err := planSplitOwnership(&shuffled)
	require.NoError(t, err)

	require.Equal(t, first.CodecOwner, second.CodecOwner)
	require.Equal(t, sortedOwnershipKeys(first.CodecOwner), first.CodecOwnerKeys)

	sharedRef := data.ReferencedTypes["shared"]
	alphaOnlyRef := data.ReferencedTypes["alphaOnly"]
	orphanRef := data.ReferencedTypes["orphan"]

	require.Equal(t, "alpha", first.CodecOwner[sharedRef.MarshalFunc()])
	require.Equal(t, "alpha", first.CodecOwner[sharedRef.UnmarshalFunc()])
	require.Equal(t, "alpha", first.CodecOwner[alphaOnlyRef.MarshalFunc()])
	require.Equal(t, "common", first.CodecOwner[orphanRef.MarshalFunc()])
	require.Equal(t, "common", first.CodecOwner[orphanRef.UnmarshalFunc()])
}

func TestSplitCodecWrappersAvoidRootPackageReferences(t *testing.T) {
	const rootImportPath = "example.com/project/graph"
	rootType := types.NewNamed(
		types.NewTypeName(0, types.NewPackage(rootImportPath, "graph"), "RootScalar", nil),
		types.Typ[types.String],
		nil,
	)
	ref := &config.TypeReference{
		Definition: &ast.Definition{Name: "RootScalar", Kind: ast.Scalar},
		GQL:        ast.NamedType("RootScalar", nil),
		GO:         rootType,
	}

	marshalKey := ref.MarshalFunc()
	unmarshalKey := ref.UnmarshalFunc()
	ownership := &splitOwnershipPlanner{
		CodecOwner: map[string]string{
			marshalKey:   "alpha",
			unmarshalKey: "alpha",
		},
		CodecOwnerKeys: []string{marshalKey, unmarshalKey},
	}

	outPath := filepath.Join(t.TempDir(), "alpha.generated.go")
	err := templates.Render(templates.Options{
		PackageName: "alpha",
		Template:    splitShardTemplate + "\n" + splitFieldsTemplate + "\n" + splitArgsTemplate + "\n" + splitDirectivesTemplate + "\n" + splitComplexityTemplate + "\n" + splitInputsTemplate + "\n" + splitCodecsTemplate,
		Filename:    outPath,
		Data: splitShardTemplateData{
			Data:             &Data{Config: &config.Config{}},
			Scope:            "scope",
			ShardName:        "alpha",
			Ownership:        ownership,
			FieldByLookupKey: map[string]*Field{},
			InputByName:      map[string]*Object{},
			CodecByFunc: map[string]*config.TypeReference{
				marshalKey:   ref,
				unmarshalKey: ref,
			},
		},
		Packages: internalcode.NewPackages(),
	})
	require.NoError(t, err)

	contents, err := os.ReadFile(outPath)
	require.NoError(t, err)

	text := string(contents)
	require.NotContains(t, text, rootImportPath)
	require.Contains(t, text, fmt.Sprintf("func %s(ctx context.Context, ec shardruntime.ObjectExecutionContext, sel ast.SelectionSet, value any) graphql.Marshaler", marshalKey))
	require.Contains(t, text, fmt.Sprintf("func %s(ctx context.Context, ec shardruntime.ObjectExecutionContext, value any) (any, error)", unmarshalKey))
}

func TestSplitShardObjectHandlersAvoidRootPackageReferences(t *testing.T) {
	const rootImportPath = "example.com/project/graph"
	rootObjectType := types.NewNamed(
		types.NewTypeName(0, types.NewPackage(rootImportPath, "graph"), "User", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	object := &Object{
		Definition: &ast.Definition{Name: "User", Kind: ast.Object},
		Type:       rootObjectType,
	}

	outPath := filepath.Join(t.TempDir(), "schema.generated.go")
	err := templates.Render(templates.Options{
		PackageName: "alpha",
		Template:    splitShardTemplate + "\n" + splitFieldsTemplate + "\n" + splitArgsTemplate + "\n" + splitDirectivesTemplate + "\n" + splitComplexityTemplate + "\n" + splitInputsTemplate + "\n" + splitCodecsTemplate,
		Filename:    outPath,
		Data: splitShardTemplateData{
			Data: &Data{
				Config:  &config.Config{},
				Objects: Objects{object},
			},
			Scope:            "scope",
			ShardName:        "alpha",
			Ownership:        &splitOwnershipPlanner{},
			FieldByLookupKey: map[string]*Field{},
			InputByName:      map[string]*Object{},
			CodecByFunc:      map[string]*config.TypeReference{},
		},
		Packages: internalcode.NewPackages(),
	})
	require.NoError(t, err)

	contents, err := os.ReadFile(outPath)
	require.NoError(t, err)

	text := string(contents)
	require.NotContains(t, text, rootImportPath)
	require.Contains(t, text, "return _User(ctx, ec, sel, obj)")
	require.Contains(t, text, "func _User(ctx context.Context, ec shardruntime.ObjectExecutionContext, sel ast.SelectionSet, obj any) graphql.Marshaler")
	require.NotContains(t, text, "typedObj, ok := obj.(")
}

func TestSplitInputWrappersAvoidRootPackageReferences(t *testing.T) {
	const rootImportPath = "example.com/project/graph"
	rootInputType := types.NewNamed(
		types.NewTypeName(0, types.NewPackage(rootImportPath, "graph"), "UserInput", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	input := &Object{
		Definition: &ast.Definition{Name: "UserInput", Kind: ast.InputObject},
		Type:       rootInputType,
	}
	ownership := &splitOwnershipPlanner{
		InputOwner: map[string]string{
			input.Name: "alpha",
		},
		InputOwnerKeys: []string{input.Name},
	}

	outPath := filepath.Join(t.TempDir(), "inputs.generated.go")
	err := templates.Render(templates.Options{
		PackageName: "alpha",
		Template:    splitInputsTemplate + "\n{{ template \"split_inputs_.gotpl\" . }}",
		Filename:    outPath,
		Data: splitShardTemplateData{
			Data:      &Data{Config: &config.Config{}},
			ShardName: "alpha",
			Ownership: ownership,
			InputByName: map[string]*Object{
				input.Name: input,
			},
		},
		Packages: internalcode.NewPackages(),
	})
	require.NoError(t, err)

	contents, err := os.ReadFile(outPath)
	require.NoError(t, err)

	text := string(contents)
	require.NotContains(t, text, rootImportPath)
	require.Contains(t, text, "func __splitInput_UserInput(_ context.Context, obj any) (any, error)")
}

func TestSplitRootUsesLookupField(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	generated, ok := snapshot[filepath.Join("graph", "generated.go")]
	require.True(t, ok)

	contents := string(generated)
	resolveFieldStart := strings.Index(contents, "func (ec *executionContext) ResolveField")
	require.NotEqual(t, -1, resolveFieldStart)

	resolveStreamFieldStart := strings.Index(contents, "func (ec *executionContext) ResolveStreamField")
	require.NotEqual(t, -1, resolveStreamFieldStart)
	require.Greater(t, resolveStreamFieldStart, resolveFieldStart)

	resolveFieldBody := contents[resolveFieldStart:resolveStreamFieldStart]
	require.Contains(t, resolveFieldBody, "shardruntime.LookupField(")
	require.Contains(t, resolveFieldBody, "objectName, fieldName")
	require.Contains(t, resolveFieldBody, "return handler(ctx, ec, field, obj)")
	require.NotContains(t, resolveFieldBody, "switch objectName+\".\"+fieldName")
	require.NotContains(t, resolveFieldBody, "switch objectName + \".\" + fieldName")
	require.Contains(t, resolveFieldBody, "panic(fmt.Sprintf(\"unknown field %s.%s\", objectName, fieldName))")
}

func TestSplitRootUsesLookupStreamField(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	generated, ok := snapshot[filepath.Join("graph", "generated.go")]
	require.True(t, ok)

	contents := string(generated)
	resolveStreamFieldStart := strings.Index(contents, "func (ec *executionContext) ResolveStreamField")
	require.NotEqual(t, -1, resolveStreamFieldStart)

	resolveStreamFieldBody := contents[resolveStreamFieldStart:]
	require.Contains(t, resolveStreamFieldBody, "shardruntime.LookupStreamField(")
	require.Contains(t, resolveStreamFieldBody, "objectName, fieldName")
	require.Contains(t, resolveStreamFieldBody, "return handler(ctx, ec, field, nil)")
	require.NotContains(t, resolveStreamFieldBody, "switch objectName+\".\"+fieldName")
	require.NotContains(t, resolveStreamFieldBody, "switch objectName + \".\" + fieldName")
	require.Contains(t, resolveStreamFieldBody, "panic(fmt.Sprintf(\"unknown stream field %s.%s\", objectName, fieldName))")
}

func TestSplitRootSeparatesStreamResolversFromRegularResolvers(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	schemaPath := filepath.Join(workDir, "graph", "subscription.graphqls")
	require.NoError(t, os.WriteFile(schemaPath, []byte("type Subscription { tick: String! }\n"), 0o644))
	t.Cleanup(func() {
		_ = os.Remove(schemaPath)
	})

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	// Root generated.go should no longer contain inline executable field/stream resolver maps.
	generated, ok := snapshot[filepath.Join("graph", "generated.go")]
	require.True(t, ok)
	contents := string(generated)
	require.NotContains(t, contents, "var splitExecutableFieldResolvers")
	require.NotContains(t, contents, "var splitExecutableStreamFieldResolvers")

	// Stream fields should be registered via RegisterStreamField in shard code,
	// while regular fields use RegisterField.
	var foundRegisterField bool
	var foundRegisterStreamField bool
	for relPath, shardContents := range snapshot {
		if !strings.HasPrefix(relPath, filepath.Join("graph", "internal", "gqlgenexec", "shards")) {
			continue
		}
		text := string(shardContents)
		if strings.Contains(text, "RegisterField(splitScope,") {
			foundRegisterField = true
			// Regular field registrations should not include subscription fields
			require.NotContains(t, text, `RegisterField(splitScope, "Subscription"`)
		}
		if strings.Contains(text, "RegisterStreamField(splitScope,") {
			foundRegisterStreamField = true
			require.Contains(t, text, `RegisterStreamField(splitScope, "Subscription", "tick"`)
		}
	}

	require.True(t, foundRegisterField, "expected shard to register regular fields via RegisterField")
	require.True(t, foundRegisterStreamField, "expected shard to register stream fields via RegisterStreamField")
}

func TestSplitComplexityLookupParity(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	generated, ok := snapshot[filepath.Join("graph", "generated.go")]
	require.True(t, ok)

	contents := string(generated)
	complexityStart := strings.Index(contents, "func (e *executableSchema) Complexity")
	require.NotEqual(t, -1, complexityStart)

	execStart := strings.Index(contents, "func (e *executableSchema) Exec")
	require.NotEqual(t, -1, execStart)
	require.Greater(t, execStart, complexityStart)

	complexityBody := contents[complexityStart:execStart]
	require.Contains(t, complexityBody, "shardruntime.LookupComplexity(")
	require.Contains(t, complexityBody, "typeName, field")
	require.Contains(t, complexityBody, "return handler(ctx, &ec, childComplexity, rawArgs)")
	require.NotContains(t, complexityBody, "switch typeName+\".\"+field")
	require.NotContains(t, complexityBody, "switch typeName + \".\" + field")
	require.Contains(t, complexityBody, "return 0, false")

	var foundRegisterComplexity bool
	for relPath, shardContents := range snapshot {
		if !strings.HasSuffix(relPath, filepath.Join("register.generated.go")) {
			continue
		}

		registerBody := string(shardContents)
		if strings.Contains(registerBody, "RegisterComplexity(splitScope,") {
			foundRegisterComplexity = true
			require.Contains(t, registerBody, "return __splitComplexity_")
		}
	}

	require.True(t, foundRegisterComplexity, "expected shard register output to include complexity registrations")
}

func TestSplitRootInputMapFromGeneratedUnmarshalers(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	generated, ok := snapshot[filepath.Join("graph", "generated.go")]
	require.True(t, ok)

	contents := string(generated)
	execStart := strings.Index(contents, "func (e *executableSchema) Exec")
	require.NotEqual(t, -1, execStart)

	executionContextStart := strings.Index(contents, "type executionContext struct")
	require.NotEqual(t, -1, executionContextStart)
	require.Greater(t, executionContextStart, execStart)

	execBody := contents[execStart:executionContextStart]
	require.Contains(t, execBody, "inputUnmarshalMap := graphql.BuildUnmarshalerMap(")
	require.NotContains(t, execBody, "shardruntime.InputUnmarshalMap(")
}

func TestSplitRuntimeIsThin(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	runtimeFile, ok := snapshot[filepath.Join("graph", "split_runtime.generated.go")]
	require.True(t, ok)

	contents := string(runtimeFile)
	require.Contains(t, contents, "intentionally thin for split-packages")
	require.NotContains(t, contents, "switch typeName + \".\" + field")
	require.NotContains(t, contents, "switch objectName + \".\" + fieldName")
	require.NotContains(t, contents, "func (ec *executionContext) _Query_hello")
}

func TestSplitImportGraphAcyclic(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	const splitScopePrefix = `const splitScope = "`
	rootImportPath := ""
	for _, contents := range snapshot {
		text := string(contents)
		start := strings.Index(text, splitScopePrefix)
		if start < 0 {
			continue
		}

		start += len(splitScopePrefix)
		end := strings.IndexByte(text[start:], '"')
		if end < 0 {
			continue
		}

		rootImportPath = text[start : start+end]
		break
	}
	require.NotEmpty(t, rootImportPath, "expected splitScope constant in generated split outputs")

	shardPrefix := rootImportPath + "/internal/gqlgenexec/shards/"
	graph := map[string]map[string]struct{}{}

	addNode := func(node string) {
		if node == "" {
			return
		}
		if _, ok := graph[node]; !ok {
			graph[node] = map[string]struct{}{}
		}
	}

	addEdge := func(from, to string) {
		if from == "" || to == "" {
			return
		}
		addNode(from)
		addNode(to)
		graph[from][to] = struct{}{}
	}

	parseImports := func(relPath string, contents []byte) []string {
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), relPath, contents, parser.ImportsOnly)
		require.NoError(t, parseErr)

		imports := make([]string, 0, len(parsed.Imports))
		for _, imp := range parsed.Imports {
			path, unquoteErr := strconv.Unquote(imp.Path.Value)
			require.NoError(t, unquoteErr)
			imports = append(imports, path)
		}
		return imports
	}

	for relPath, contents := range snapshot {
		slashPath := filepath.ToSlash(relPath)
		if !strings.HasSuffix(slashPath, ".go") {
			continue
		}

		pkgImport := ""
		if strings.HasPrefix(slashPath, "graph/internal/gqlgenexec/shards/") {
			parts := strings.Split(slashPath, "/")
			require.GreaterOrEqual(t, len(parts), 6)
			pkgImport = shardPrefix + parts[4]
		} else if strings.HasPrefix(slashPath, "graph/") {
			pkgImport = rootImportPath
		} else {
			continue
		}

		addNode(pkgImport)
		for _, imp := range parseImports(relPath, contents) {
			if imp == rootImportPath {
				require.NotEqual(t, pkgImport, rootImportPath, "root package self-import is invalid in generated output")
				require.False(t, strings.HasPrefix(pkgImport, shardPrefix), "shard package %s must not import root package %s", pkgImport, rootImportPath)
				addEdge(pkgImport, imp)
				continue
			}

			if strings.HasPrefix(imp, shardPrefix) {
				require.False(t, strings.HasPrefix(pkgImport, shardPrefix), "shard package %s must not import shard package %s", pkgImport, imp)
				addEdge(pkgImport, imp)
			}
		}
	}

	addNode(rootImportPath)
	require.NotEmpty(t, graph[rootImportPath], "expected root split package to import generated shard packages")

	state := map[string]int{}
	stack := make([]string, 0, len(graph))
	var visit func(string)
	visit = func(node string) {
		switch state[node] {
		case 1:
			cycle := append(append([]string(nil), stack...), node)
			require.Failf(t, "split import graph contains cycle", "cycle: %s", strings.Join(cycle, " -> "))
			return
		case 2:
			return
		}

		state[node] = 1
		stack = append(stack, node)

		next := make([]string, 0, len(graph[node]))
		for dep := range graph[node] {
			next = append(next, dep)
		}
		sort.Strings(next)
		for _, dep := range next {
			visit(dep)
		}

		stack = stack[:len(stack)-1]
		state[node] = 2
	}

	nodes := make([]string, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		visit(node)
	}
}

func TestSplitGeneratedImportGraphAcyclic(t *testing.T) {
	TestSplitImportGraphAcyclic(t)
}

func TestSplitShardFieldArgsEmission(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	var foundFieldTemplateEmission bool
	var foundArgsTemplateEmission bool
	for relPath, contents := range snapshot {
		if !strings.HasPrefix(relPath, filepath.Join("graph", "internal", "gqlgenexec", "shards")) {
			continue
		}

		text := string(contents)
		if strings.Contains(text, "split_fields_.gotpl") && strings.Contains(text, "func __splitField_") {
			foundFieldTemplateEmission = true
		}
		if strings.Contains(text, "split_args_.gotpl") && strings.Contains(text, "func __splitArgs_") {
			foundArgsTemplateEmission = true
		}
	}

	require.True(t, foundFieldTemplateEmission, "expected split shard field emission from split_fields_.gotpl")
	require.True(t, foundArgsTemplateEmission, "expected split shard args emission from split_args_.gotpl")
}

func TestSplitComplexityEmissionByOwner(t *testing.T) {
	newObject := func(name, sourceFile, fieldName string) *Object {
		object := &Object{
			Definition: &ast.Definition{
				Name:     name,
				Kind:     ast.Object,
				Position: &ast.Position{Src: &ast.Source{Name: sourceFile}},
			},
		}
		field := &Field{FieldDefinition: &ast.FieldDefinition{Name: fieldName}, Object: object}
		object.Fields = []*Field{field}
		return object
	}

	data := &Data{
		Config: &config.Config{Exec: config.ExecConfig{FilenameTemplate: "{name}.generated.go"}},
		Objects: Objects{
			newObject("AlphaQuery", "graph/alpha.graphqls", "alphaField"),
			newObject("ZetaQuery", "graph/zeta.graphqls", "zetaField"),
		},
	}
	ownership, err := planSplitOwnership(data)
	require.NoError(t, err)

	outDir := t.TempDir()
	alphaPath := filepath.Join(outDir, "alpha_complexity.generated.go")
	zetaPath := filepath.Join(outDir, "zeta_complexity.generated.go")

	err = templates.Render(templates.Options{
		PackageName: "splitcomplexitytest",
		Template:    splitComplexityTemplate + "\n{{ template \"split_complexity_.gotpl\" . }}",
		Filename:    alphaPath,
		Data: splitShardTemplateData{
			Data:             data,
			Scope:            "scope",
			ShardName:        "alpha",
			Ownership:        ownership,
			FieldByLookupKey: buildFieldLookupMap(data),
		},
		Packages: internalcode.NewPackages(),
	})
	require.NoError(t, err)

	err = templates.Render(templates.Options{
		PackageName: "splitcomplexitytest",
		Template:    splitComplexityTemplate + "\n{{ template \"split_complexity_.gotpl\" . }}",
		Filename:    zetaPath,
		Data: splitShardTemplateData{
			Data:             data,
			Scope:            "scope",
			ShardName:        "zeta",
			Ownership:        ownership,
			FieldByLookupKey: buildFieldLookupMap(data),
		},
		Packages: internalcode.NewPackages(),
	})
	require.NoError(t, err)

	alphaContents, err := os.ReadFile(alphaPath)
	require.NoError(t, err)
	alphaText := string(alphaContents)
	require.Contains(t, alphaText, "split_complexity_.gotpl")
	require.Contains(t, alphaText, "func __splitComplexity_AlphaQuery_alphaField")
	require.Contains(t, alphaText, "return ec.ResolveExecutableComplexity(ctx, \"AlphaQuery\", \"alphaField\", childComplexity, rawArgs)")
	require.NotContains(t, alphaText, "func __splitComplexity_ZetaQuery_zetaField")

	zetaContents, err := os.ReadFile(zetaPath)
	require.NoError(t, err)
	zetaText := string(zetaContents)
	require.Contains(t, zetaText, "split_complexity_.gotpl")
	require.Contains(t, zetaText, "func __splitComplexity_ZetaQuery_zetaField")
	require.Contains(t, zetaText, "return ec.ResolveExecutableComplexity(ctx, \"ZetaQuery\", \"zetaField\", childComplexity, rawArgs)")
	require.NotContains(t, zetaText, "func __splitComplexity_AlphaQuery_alphaField")
}

func TestSplitDirectiveOrderParity(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(workDir, "graph", "directive_order.graphqls"))
	})

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "graph", "directive_order.graphqls"), []byte(`directive @first on FIELD_DEFINITION
directive @second on FIELD_DEFINITION

extend type Query {
  pong: String! @first @second
}
`), 0o644))

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	var directiveShard string
	for relPath, contents := range snapshot {
		if !strings.HasPrefix(relPath, filepath.Join("graph", "internal", "gqlgenexec", "shards")) {
			continue
		}

		text := string(contents)
		if strings.Contains(text, "split_directives_.gotpl") && strings.Contains(text, "func __splitDirectives_Query_pong") {
			directiveShard = text
			break
		}
	}

	require.NotEmpty(t, directiveShard, "expected split shard directive emission from split_directives_.gotpl")
	require.Contains(t, directiveShard, "directive0 := next")
	require.Contains(t, directiveShard, "return directive0(ctx)")
	require.Contains(t, directiveShard, "return directive1(ctx)")

	firstPos := strings.Index(directiveShard, "// directive first")
	secondPos := strings.Index(directiveShard, "// directive second")
	require.NotEqual(t, -1, firstPos)
	require.NotEqual(t, -1, secondPos)
	require.Less(t, firstPos, secondPos)
}

func TestSplitRegistrationOrderDeterministic(t *testing.T) {
	workDir := chdirToLocalSplitFixtureWorkspace(t)

	cleanupSplitGeneratedFiles(workDir)
	snapshot := generateSplitSnapshot(t)

	registerCount := 0
	for relPath, contents := range snapshot {
		if !strings.HasSuffix(relPath, filepath.Join("register.generated.go")) {
			continue
		}

		registerCount++
		registrations := splitRegistrationOrder(string(contents))
		if len(registrations) == 0 {
			continue
		}

		sortedRegistrations := append([]string(nil), registrations...)
		sort.Strings(sortedRegistrations)
		require.Equal(t, sortedRegistrations, registrations, "expected deterministic (object, field) registration order in %s", relPath)
	}

	require.Greater(t, registerCount, 0, "expected at least one register.generated.go file in split shards")
}

func TestSplitInputRegistrationEmission(t *testing.T) {
	data := splitInputOwnerTestData()
	for _, object := range data.Objects {
		for _, field := range object.Fields {
			for _, arg := range field.Args {
				if arg.TypeReference != nil {
					if arg.TypeReference.GQL == nil {
						arg.TypeReference.GQL = ast.NonNullNamedType(arg.TypeReference.Definition.Name, nil)
					}
					if arg.TypeReference.GO == nil {
						arg.TypeReference.GO = types.Typ[types.Int]
					}
				}
			}
		}
	}
	for _, input := range data.Inputs {
		input.Type = types.NewNamed(
			types.NewTypeName(0, types.NewPackage("github.com/99designs/gqlgen/codegen/testinput", "testinput"), input.Name, nil),
			types.NewStruct(nil, nil),
			nil,
		)
		for _, field := range input.Fields {
			if field.TypeReference != nil {
				if field.TypeReference.GQL == nil {
					field.TypeReference.GQL = ast.NonNullNamedType(field.TypeReference.Definition.Name, nil)
				}
				if field.TypeReference.GO == nil {
					field.TypeReference.GO = types.Typ[types.Int]
				}
			}
		}
	}

	ownership, err := planSplitOwnership(data)
	require.NoError(t, err)

	builds := map[string]*Data{}
	require.NoError(t, addObjects(data, &builds))

	var alphaBuild *Data
	for filename, build := range builds {
		if filename == "" || build == nil || len(build.Objects) == 0 {
			continue
		}
		alphaBuild = build
		break
	}
	require.NotNil(t, alphaBuild)
	require.Empty(t, alphaBuild.Inputs)
	require.Empty(t, buildInputLookupMap(alphaBuild), "object-only shard builds do not include input definitions")

	templateData := splitShardTemplateData{
		Data:             alphaBuild,
		Scope:            "scope",
		ShardName:        "alpha",
		Ownership:        ownership,
		FieldByLookupKey: buildFieldLookupMap(alphaBuild),
		InputByName:      buildInputLookupMap(data),
	}

	outDir := t.TempDir()
	inputsPath := filepath.Join(outDir, "inputs.generated.go")
	registerPath := filepath.Join(outDir, "register.generated.go")

	err = templates.Render(templates.Options{
		PackageName: "splitinputstest",
		Template:    splitInputsTemplate + "\n{{ template \"split_inputs_.gotpl\" . }}",
		Filename:    inputsPath,
		Data:        templateData,
		Packages:    internalcode.NewPackages(),
	})
	require.NoError(t, err)

	err = templates.Render(templates.Options{
		PackageName: "splitinputstest",
		Template:    splitRegisterTemplate,
		Filename:    registerPath,
		Data:        templateData,
		Packages:    internalcode.NewPackages(),
	})
	require.NoError(t, err)

	inputsContents, err := os.ReadFile(inputsPath)
	require.NoError(t, err)
	inputsText := string(inputsContents)
	require.Contains(t, inputsText, "split_inputs_.gotpl")
	require.Contains(t, inputsText, "func __splitInput_NestedInput")
	require.Contains(t, inputsText, "func __splitInput_SharedInput")

	registerContents, err := os.ReadFile(registerPath)
	require.NoError(t, err)
	registerText := string(registerContents)
	require.Contains(t, registerText, "RegisterInputUnmarshaler(splitScope, \"NestedInput\"")
	require.Contains(t, registerText, "RegisterInputUnmarshaler(splitScope, \"SharedInput\"")
	require.NotContains(t, registerText, "RegisterInputUnmarshaler(splitScope, \"OrphanInput\"")

}

var splitRegistrationPattern = regexp.MustCompile(`Register(?:Stream)?Field\(splitScope,\s*"([^"]+)",\s*"([^"]+)"`)

func splitRegistrationOrder(contents string) []string {
	matches := splitRegistrationPattern.FindAllStringSubmatch(contents, -1)
	order := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		order = append(order, match[1]+"."+match[2])
	}

	return order
}

func generateSplitSnapshot(t *testing.T) map[string][]byte {
	t.Helper()

	data := buildSplitData(t)

	err := GenerateCode(data)
	require.NoError(t, err)

	workDir, err := os.Getwd()
	require.NoError(t, err)

	files := collectSplitGeneratedFiles(t, workDir)
	require.NotEmpty(t, files)

	snapshot := make(map[string][]byte, len(files))
	for _, rel := range files {
		fullPath := filepath.Join(workDir, rel)
		contents, readErr := os.ReadFile(fullPath)
		require.NoError(t, readErr)
		snapshot[rel] = contents
	}

	return snapshot
}

func buildSplitData(t *testing.T) *Data {
	t.Helper()

	cfg, err := config.LoadConfigFromDefaultLocations()
	require.NoError(t, err)
	require.NoError(t, cfg.LoadSchema())
	require.NoError(t, cfg.LoadSchema())

	ClearInlineArgsMetadata()
	require.NoError(t, ExpandInlineArguments(cfg.Schema))
	require.NoError(t, cfg.Init())

	data, err := BuildData(cfg)
	require.NoError(t, err)
	return data
}

func shuffledOwnershipData(data *Data) *Data {
	shuffled := *data
	objects := make(Objects, len(data.Objects))
	for i := range data.Objects {
		obj := *data.Objects[len(data.Objects)-1-i]
		fields := make([]*Field, len(obj.Fields))
		for j := range obj.Fields {
			fields[j] = obj.Fields[len(obj.Fields)-1-j]
		}
		obj.Fields = fields
		objects[i] = &obj
	}
	shuffled.Objects = objects
	return &shuffled
}

func splitInputOwnerTestData() *Data {
	sharedInputDef := &ast.Definition{Name: "SharedInput", Kind: ast.InputObject}
	nestedInputDef := &ast.Definition{Name: "NestedInput", Kind: ast.InputObject}
	orphanInputDef := &ast.Definition{Name: "OrphanInput", Kind: ast.InputObject}

	sharedInput := &Object{
		Definition: sharedInputDef,
		Fields: []*Field{
			{
				FieldDefinition: &ast.FieldDefinition{Name: "nested"},
				TypeReference:   &config.TypeReference{Definition: nestedInputDef},
			},
		},
	}
	nestedInput := &Object{Definition: nestedInputDef}
	orphanInput := &Object{Definition: orphanInputDef}

	newObject := func(name, sourceFile, fieldName string) *Object {
		object := &Object{
			Definition: &ast.Definition{
				Name: name,
				Kind: ast.Object,
				Position: &ast.Position{
					Src: &ast.Source{Name: sourceFile},
				},
			},
		}

		field := &Field{
			FieldDefinition: &ast.FieldDefinition{Name: fieldName},
			Args: []*FieldArgument{
				{
					ArgumentDefinition: &ast.ArgumentDefinition{Name: "input"},
					TypeReference:      &config.TypeReference{Definition: sharedInputDef},
				},
			},
			Object: object,
		}
		object.Fields = []*Field{field}

		return object
	}

	return &Data{
		Config: &config.Config{Exec: config.ExecConfig{FilenameTemplate: "{name}.generated.go"}},
		Objects: Objects{
			newObject("AlphaQuery", "graph/alpha.graphqls", "alphaField"),
			newObject("ZetaQuery", "graph/zeta.graphqls", "zetaField"),
		},
		Inputs: Objects{sharedInput, nestedInput, orphanInput},
	}
}

func splitCodecOwnerTestData() *Data {
	newTypeRef := func(defName string, goType types.Type) *config.TypeReference {
		def := &ast.Definition{Name: defName, Kind: ast.Scalar}
		return &config.TypeReference{
			Definition: def,
			GQL:        ast.NonNullNamedType(defName, nil),
			GO:         goType,
		}
	}

	shared := newTypeRef("SharedCodec", types.Typ[types.String])
	alphaOnly := newTypeRef("AlphaCodec", types.Typ[types.Int])
	orphan := newTypeRef("OrphanCodec", types.Typ[types.Bool])

	newObject := func(name, sourceFile string, fields ...*Field) *Object {
		object := &Object{
			Definition: &ast.Definition{
				Name: name,
				Kind: ast.Object,
				Position: &ast.Position{
					Src: &ast.Source{Name: sourceFile},
				},
			},
		}
		for _, field := range fields {
			field.Object = object
		}
		object.Fields = fields
		return object
	}

	newField := func(name string, returnType, argType *config.TypeReference) *Field {
		field := &Field{
			FieldDefinition: &ast.FieldDefinition{Name: name},
			TypeReference:   returnType,
		}
		if argType != nil {
			field.Args = []*FieldArgument{{
				ArgumentDefinition: &ast.ArgumentDefinition{Name: "input"},
				TypeReference:      argType,
			}}
		}
		return field
	}

	alpha := newObject(
		"AlphaQuery",
		"graph/alpha.graphqls",
		newField("shared", shared, shared),
		newField("alphaOnly", alphaOnly, nil),
	)
	zeta := newObject(
		"ZetaQuery",
		"graph/zeta.graphqls",
		newField("shared", shared, shared),
	)

	return &Data{
		Config: &config.Config{Exec: config.ExecConfig{FilenameTemplate: "{name}.generated.go"}},
		Objects: Objects{
			alpha,
			zeta,
		},
		ReferencedTypes: map[string]*config.TypeReference{
			"shared":    shared,
			"alphaOnly": alphaOnly,
			"orphan":    orphan,
		},
	}
}

func collectSplitGeneratedFiles(t *testing.T, workDir string) []string {
	t.Helper()

	generatedHeader := []byte("// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.")

	files := []string{
		filepath.Join("graph", "generated.go"),
		filepath.Join("graph", "split_runtime.generated.go"),
	}

	imports, err := filepath.Glob(filepath.Join(workDir, "graph", "split_shard_import_*.generated.go"))
	require.NoError(t, err)
	for _, match := range imports {
		rel, relErr := filepath.Rel(workDir, match)
		require.NoError(t, relErr)
		files = append(files, rel)
	}

	shardRoot := filepath.Join(workDir, "graph", "internal", "gqlgenexec")
	err = filepath.WalkDir(shardRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !bytes.HasPrefix(contents, generatedHeader) {
			return nil
		}

		rel, relErr := filepath.Rel(workDir, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, rel)
		return nil
	})
	require.NoError(t, err)

	sort.Strings(files)
	return files
}

func cleanupSplitGeneratedFiles(workDir string) {
	_ = os.Remove(filepath.Join(workDir, "graph", "generated.go"))
	_ = os.Remove(filepath.Join(workDir, "graph", "split_runtime.generated.go"))
	for i := range 64 {
		_ = os.Remove(filepath.Join(workDir, "graph", fmt.Sprintf("split_shard_import_%d.generated.go", i)))
	}
	_ = os.RemoveAll(filepath.Join(workDir, "graph", "internal", "gqlgenexec"))
}

func chdirToLocalSplitFixtureWorkspace(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	fixturesRoot := filepath.Join(wd, "testserver")
	fixtureDir := filepath.Join(wd, "..", "api", "testdata", "splitpackages")

	workDir, err := os.MkdirTemp(fixturesRoot, "splitpackages-work-")
	require.NoError(t, err)
	require.NoError(t, copySplitFixtureWorkspace(fixtureDir, workDir))

	t.Chdir(workDir)
	t.Cleanup(func() {
		cleanupSplitGeneratedFiles(workDir)
		t.Chdir(wd)
		_ = os.RemoveAll(workDir)
	})

	return workDir
}

func copySplitFixtureWorkspace(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fileInfo, err := d.Info()
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, contents, fileInfo.Mode().Perm())
	})
}
