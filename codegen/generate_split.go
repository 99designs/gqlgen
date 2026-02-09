package codegen

import (
	_ "embed"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	internalcode "github.com/99designs/gqlgen/internal/code"
)

//go:embed split_root_.gotpl
var splitRootTemplate string

//go:embed split_shard_.gotpl
var splitShardTemplate string

//go:embed split_imports_.gotpl
var splitImportsTemplate string

type splitRootTemplateData struct {
	*Data
	Scope string
}

type splitShardTemplateData struct {
	*Data
	Scope string
}

type splitImportsTemplateData struct {
	Import string
}

func generateSplitPackages(data *Data) error {
	if err := cleanupSplitRootImports(data); err != nil {
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

func cleanupSplitRootImports(data *Data) error {
	pattern := filepath.Join(data.Config.Exec.Dir(), "split_shard_import_*.generated.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid split import cleanup glob %q: %w", pattern, err)
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale split import file %q: %w", match, err)
		}
	}
	return nil
}

func generateSplitRootGateway(data *Data, scope string) error {
	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Template:        splitRootTemplate,
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
		Filename:        path,
		Data:            data,
		RegionTags:      true,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      codegenTemplates,
	})
}

func generateSplitShardPackages(data *Data, scope string) ([]string, error) {
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

		pkg := internalcode.NameForDir(shardDir)
		if pkg == "" {
			pkg = shardName
		}
		build.Config.Exec.Package = pkg

		if err := templates.Render(templates.Options{
			PackageName:     pkg,
			Template:        splitShardTemplate,
			Filename:        shardPath,
			Data:            splitShardTemplateData{Data: build, Scope: scope},
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
	return data.Config.Exec.Package + ":" + filepath.Base(data.Config.Exec.Filename)
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
	return name
}

func splitShortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum32())[:6]
}
