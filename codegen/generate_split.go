package codegen

import (
	_ "embed"
	"fmt"
	"go/token"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	internalcode "github.com/99designs/gqlgen/internal/code"
)

//go:embed split_root_.gotpl
var splitRootTemplate string

//go:embed split_shard_.gotpl
var splitShardTemplate string

//go:embed split_fields_.gotpl
var splitFieldsTemplate string

//go:embed split_args_.gotpl
var splitArgsTemplate string

//go:embed split_directives_.gotpl
var splitDirectivesTemplate string

//go:embed split_complexity_.gotpl
var splitComplexityTemplate string

//go:embed split_inputs_.gotpl
var splitInputsTemplate string

//go:embed split_codecs_.gotpl
var splitCodecsTemplate string

//go:embed split_register_.gotpl
var splitRegisterTemplate string

//go:embed split_imports_.gotpl
var splitImportsTemplate string

//go:embed split_runtime_.gotpl
var splitRuntimeTemplate string

//go:embed args.gotpl
var argsTemplate string

//go:embed directives.gotpl
var directivesTemplate string

//go:embed field.gotpl
var fieldTemplate string

//go:embed input.gotpl
var inputTemplate string

//go:embed interface.gotpl
var interfaceTemplate string

//go:embed type.gotpl
var typeTemplate string

type splitRootTemplateData struct {
	*Data
	Scope string
}

type splitShardTemplateData struct {
	*Data
	Scope            string
	ShardName        string
	Ownership        *splitOwnershipPlanner
	FieldByLookupKey map[string]*Field
	InputByName      map[string]*Object
	CodecByFunc      map[string]*config.TypeReference
}

type splitImportsTemplateData struct {
	Import string
}

func generateSplitPackages(data *Data) error {
	if err := cleanupSplitGeneratedOutputs(data); err != nil {
		return err
	}

	scope := splitScope(data)

	if err := generateSplitRootGateway(data, scope); err != nil {
		return err
	}

	if err := generateSplitRootRuntime(data); err != nil {
		return err
	}

	shardImports, err := generateSplitShardPackages(data, scope)
	if err != nil {
		return err
	}

	return generateSplitShardImports(data, shardImports)
}

func cleanupSplitGeneratedOutputs(data *Data) error {
	if err := removeSplitGeneratedByGlob(filepath.Join(data.Config.Exec.Dir(), "split_shard_import_*.generated.go"), "split import"); err != nil {
		return err
	}

	runtimePath := filepath.Join(data.Config.Exec.Dir(), "split_runtime.generated.go")
	if err := os.Remove(runtimePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale split runtime file %q: %w", runtimePath, err)
	}

	generatedShardFiles, err := listSplitShardGeneratedFiles(data.Config.Exec.ShardDir, data.Config.Exec.ShardFilenameTemplate)
	if err != nil {
		return err
	}
	for _, path := range generatedShardFiles {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale split shard file %q: %w", path, err)
		}
	}

	if err := pruneEmptyDirs(data.Config.Exec.ShardDir); err != nil {
		return err
	}

	return nil
}

func removeSplitGeneratedByGlob(pattern string, kind string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid %s cleanup glob %q: %w", kind, pattern, err)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale %s file %q: %w", kind, match, err)
		}
	}

	return nil
}

func listSplitShardGeneratedFiles(root string, shardFilenameTemplate string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat split shard root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("split shard root %q is not a directory", root)
	}

	shardFilePattern, err := compileSplitShardFilenamePattern(shardFilenameTemplate)
	if err != nil {
		return nil, err
	}

	var generated []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		owned, ownerErr := isSplitOwnedGeneratedFile(path, d.Name(), shardFilePattern)
		if ownerErr != nil {
			return ownerErr
		}
		if owned {
			generated = append(generated, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk split shard root %q: %w", root, err)
	}

	sort.Strings(generated)
	return generated, nil
}

func compileSplitShardFilenamePattern(shardFilenameTemplate string) (*regexp.Regexp, error) {
	if shardFilenameTemplate == "" {
		shardFilenameTemplate = "{name}.generated.go"
	}

	escaped := regexp.QuoteMeta(shardFilenameTemplate)
	escaped = strings.ReplaceAll(escaped, regexp.QuoteMeta("{name}"), "[^/]+")
	pattern, err := regexp.Compile("^" + escaped + "$")
	if err != nil {
		return nil, fmt.Errorf("compile split shard filename pattern for %q: %w", shardFilenameTemplate, err)
	}

	return pattern, nil
}

func isSplitOwnedGeneratedFile(path string, name string, shardFilePattern *regexp.Regexp) (bool, error) {
	if name == "register.generated.go" {
		return true, nil
	}

	if !shardFilePattern.MatchString(name) {
		return false, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read split shard candidate %q: %w", path, err)
	}

	if strings.Contains(string(contents), "const splitScope =") {
		return true, nil
	}

	return false, nil
}

func pruneEmptyDirs(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat split shard root for prune %q: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("split shard root %q is not a directory", root)
	}

	var dirs []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk split shard root for prune %q: %w", root, err)
	}

	sort.Slice(dirs, func(i, j int) bool {
		if len(dirs[i]) == len(dirs[j]) {
			return dirs[i] > dirs[j]
		}
		return len(dirs[i]) > len(dirs[j])
	})

	for _, dir := range dirs {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return fmt.Errorf("read split shard dir %q: %w", dir, readErr)
		}
		if len(entries) > 0 {
			continue
		}
		if removeErr := os.Remove(dir); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("remove empty split shard dir %q: %w", dir, removeErr)
		}
	}

	return nil
}

func generateSplitRootGateway(data *Data, scope string) error {
	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Template:        splitRootTemplate + "\n" + argsTemplate + "\n" + directivesTemplate + "\n" + fieldTemplate + "\n" + inputTemplate + "\n" + interfaceTemplate + "\n" + typeTemplate,
		Filename:        data.Config.Exec.Filename,
		Data:            splitRootTemplateData{Data: data, Scope: scope},
		RegionTags:      false,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      codegenTemplates,
	})
}

func generateSplitRootRuntime(data *Data) error {
	path := filepath.Join(data.Config.Exec.Dir(), "split_runtime.generated.go")
	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Template:        splitRuntimeTemplate,
		Filename:        path,
		Data:            data,
		RegionTags:      false,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      codegenTemplates,
	})
}

func generateSplitShardPackages(data *Data, scope string) ([]string, error) {
	ownership, err := planSplitOwnership(data)
	if err != nil {
		return nil, err
	}

	builds := map[string]*Data{}
	if err := addObjects(data, &builds); err != nil {
		return nil, err
	}

	var filenames []string
	for filename := range builds {
		if filename != "" {
			filenames = append(filenames, filename)
		}
	}
	sort.Strings(filenames)

	var imports []string
	usedShardNames := map[string]string{}

	for _, filename := range filenames {
		build := builds[filename]
		if build == nil || len(build.Objects) == 0 {
			continue
		}

		shardName := splitShardName(filename, build, usedShardNames)
		shardDir := filepath.Join(data.Config.Exec.ShardDir, shardName)
		shardFile := strings.ReplaceAll(data.Config.Exec.ShardFilenameTemplate, "{name}", shardName)
		shardPath := filepath.Join(shardDir, shardFile)

		if err := os.MkdirAll(shardDir, 0o755); err != nil {
			return nil, fmt.Errorf("create split shard dir %q: %w", shardDir, err)
		}

		pkg := internalcode.NameForDir(shardDir)
		if pkg == "" {
			pkg = shardName
		}
		build.Config.Exec.Package = pkg

		if err := templates.Render(templates.Options{
			PackageName: pkg,
			Template:    splitShardTemplate + "\n" + splitFieldsTemplate + "\n" + splitArgsTemplate + "\n" + splitDirectivesTemplate + "\n" + splitComplexityTemplate + "\n" + splitInputsTemplate + "\n" + splitCodecsTemplate,
			Filename:    shardPath,
			Data: splitShardTemplateData{
				Data:             build,
				Scope:            scope,
				ShardName:        shardName,
				Ownership:        ownership,
				FieldByLookupKey: buildFieldLookupMap(build),
				InputByName:      buildInputLookupMap(data),
				CodecByFunc:      buildCodecLookupMap(data),
			},
			RegionTags:      false,
			GeneratedHeader: true,
			Packages:        data.Config.Packages,
		}); err != nil {
			return nil, err
		}

		registerPath := filepath.Join(shardDir, "register.generated.go")
		if err := templates.Render(templates.Options{
			PackageName: pkg,
			Template:    splitRegisterTemplate,
			Filename:    registerPath,
			Data: splitShardTemplateData{
				Data:             build,
				Scope:            scope,
				ShardName:        shardName,
				Ownership:        ownership,
				FieldByLookupKey: buildFieldLookupMap(build),
				InputByName:      buildInputLookupMap(data),
			},
			RegionTags:      false,
			GeneratedHeader: true,
			Packages:        data.Config.Packages,
		}); err != nil {
			return nil, err
		}

		importPath := internalcode.ImportPathForDir(shardDir)
		if importPath == "" {
			return nil, fmt.Errorf("unable to determine import path for shard dir %s", shardDir)
		}
		imports = append(imports, importPath)
	}

	sort.Strings(imports)
	dedup := make([]string, 0, len(imports))
	for _, imp := range imports {
		if len(dedup) == 0 || dedup[len(dedup)-1] != imp {
			dedup = append(dedup, imp)
		}
	}

	return dedup, nil
}

func buildFieldLookupMap(data *Data) map[string]*Field {
	fieldByLookupKey := make(map[string]*Field)
	for _, object := range data.Objects {
		for _, field := range object.Fields {
			fieldByLookupKey[object.Name+"."+field.Name] = field
		}
	}

	return fieldByLookupKey
}

func buildInputLookupMap(data *Data) map[string]*Object {
	inputByName := make(map[string]*Object)
	for _, input := range data.Inputs {
		inputByName[input.Name] = input
	}

	return inputByName
}

func buildCodecLookupMap(data *Data) map[string]*config.TypeReference {
	codecByFunc := make(map[string]*config.TypeReference)
	for _, ref := range data.ReferencedTypes {
		if ref == nil {
			continue
		}

		if marshal := ref.MarshalFunc(); marshal != "" {
			codecByFunc[marshal] = ref
		}
		if unmarshal := ref.UnmarshalFunc(); unmarshal != "" {
			codecByFunc[unmarshal] = ref
		}
	}

	return codecByFunc
}

func generateSplitShardImports(data *Data, shardImports []string) error {
	if len(shardImports) == 0 {
		return nil
	}

	for i, shardImport := range shardImports {
		path := filepath.Join(data.Config.Exec.Dir(), fmt.Sprintf("split_shard_import_%d.generated.go", i))
		if err := templates.Render(templates.Options{
			PackageName:     data.Config.Exec.Package,
			Template:        splitImportsTemplate,
			Filename:        path,
			Data:            splitImportsTemplateData{Import: shardImport},
			RegionTags:      false,
			GeneratedHeader: true,
			Packages:        data.Config.Packages,
			TemplateFS:      codegenTemplates,
		}); err != nil {
			return err
		}
	}
	return nil
}

func splitScope(data *Data) string {
	if path := data.Config.Exec.ImportPath(); path != "" {
		return path
	}

	fallback := data.Config.Exec.Package + ":" + filepath.Base(data.Config.Exec.Filename)
	if data.Config.Exec.Filename == "" {
		return fallback
	}

	return fallback + ":" + splitShortHash(data.Config.Exec.Filename)
}

var splitNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func splitShardName(filename string, build *Data, used map[string]string) string {
	raw := splitRawShardName(filename, build)
	candidate := splitSanitizeName(raw)

	if prev, exists := used[candidate]; exists && prev != filename {
		candidate = candidate + "_" + splitShortHash(filename)
	}
	used[candidate] = filename
	return candidate
}

func splitRawShardName(filename string, build *Data) string {
	if len(build.Config.Sources) > 0 && build.Config.Sources[0] != nil {
		src := build.Config.Sources[0]
		name := filepath.Base(src.Name)
		ext := filepath.Ext(name)
		return strings.TrimSuffix(name, ext)
	}
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.TrimSuffix(base, ".generated")
	if base == "" {
		base = "common"
	}
	return base
}

func splitSanitizeName(name string) string {
	name = splitNameSanitizer.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	name = strings.ToLower(name)
	if name == "" {
		name = "shard"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "s_" + name
	}
	if token.Lookup(name).IsKeyword() {
		name = "s_" + name
	}
	return name
}

func splitShortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())[:6]
}
