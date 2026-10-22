package api

import (
	"errors"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"

	"golang.org/x/tools/imports"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/internal/code"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/99designs/gqlgen/plugin/resolvergen"
)

var (
	urlRegex = regexp.MustCompile(
		`(?s)@link.*\(.*url:\s*?"(.*?)"[^)]+\)`,
	) // regex to grab the url of a link directive, should it exist
	versionRegex = regexp.MustCompile(
		`v(\d+).(\d+)$`,
	) // regex to grab the version number from a url
)

// Generate generates GraphQL code based on the provided config.
func Generate(cfg *config.Config, option ...Option) error {
	return generate(cfg, nil, option...)
}

// GenerateIncremental generates code only for schemas affected by changes.
// changedSchemas should contain paths to schema files that have changed
// (e.g., from git diff). If empty, performs full generation.
// Use verbose to enable detailed logging of what's being regenerated.
func GenerateIncremental(
	cfg *config.Config,
	changedSchemas []string,
	verbose bool,
	option ...Option,
) error {
	return generate(cfg, &codegen.IncrementalOptions{
		ChangedSchemas: changedSchemas,
		Verbose:        verbose,
	}, option...)
}

// generate is the shared implementation for both Generate and GenerateIncremental.
// If incrementalOpts is nil, performs full generation. Otherwise, uses incremental generation.
func generate(
	cfg *config.Config,
	incrementalOpts *codegen.IncrementalOptions,
	option ...Option,
) error {
	_ = syscall.Unlink(cfg.Exec.Filename)
	if cfg.Model.IsDefined() {
		_ = syscall.Unlink(cfg.Model.Filename)
	}

	plugins := []plugin.Plugin{}
	if cfg.Model.IsDefined() {
		plugins = append(plugins, modelgen.New())
	}
	plugins = append(plugins, resolvergen.New())
	if cfg.Federation.IsDefined() {
		if cfg.Federation.Version == 0 { // default to using the user's choice of version, but if unset, try to sort out which federation version to use
			// check the sources, and if one is marked as federation v2, we mark the entirety to be
			// generated using that format
			for _, v := range cfg.Sources {
				cfg.Federation.Version = 1
				urlString := urlRegex.FindStringSubmatch(v.Input)
				// e.g. urlString[1] == "https://specs.apollo.dev/federation/v2.7"
				if urlString != nil {
					matches := versionRegex.FindStringSubmatch(urlString[1])
					if matches[1] == "2" {
						cfg.Federation.Version = 2
						break
					}
				}
			}
		}
		federationPlugin, err := federation.New(cfg.Federation.Version, cfg)
		if err != nil {
			return fmt.Errorf("failed to construct the Federation plugin: %w", err)
		}
		plugins = append([]plugin.Plugin{federationPlugin}, plugins...)
	}

	for _, o := range option {
		o(cfg, &plugins)
	}

	if cfg.LocalPrefix != "" {
		imports.LocalPrefix = cfg.LocalPrefix
	}

	for _, p := range plugins {
		//nolint:staticcheck // for backwards compatibility only
		if inj, ok := p.(plugin.EarlySourceInjector); ok {
			if s := inj.InjectSourceEarly(); s != nil {
				cfg.Sources = append(cfg.Sources, s)
			}
		}
		if inj, ok := p.(plugin.EarlySourcesInjector); ok {
			s, err := inj.InjectSourcesEarly()
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name(), err)
			}
			cfg.Sources = append(cfg.Sources, s...)
		}
	}

	if err := cfg.LoadSchema(); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	for _, p := range plugins {
		if inj, ok := p.(plugin.LateSourceInjector); ok {
			if s := inj.InjectSourceLate(cfg.Schema); s != nil {
				cfg.Sources = append(cfg.Sources, s)
			}
		}
		if inj, ok := p.(plugin.LateSourcesInjector); ok {
			s, err := inj.InjectSourcesLate(cfg.Schema)
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name(), err)
			}
			cfg.Sources = append(cfg.Sources, s...)
		}
	}

	// LoadSchema again now we have everything
	if err := cfg.LoadSchema(); err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	codegen.ClearInlineArgsMetadata()
	if err := codegen.ExpandInlineArguments(cfg.Schema); err != nil {
		return fmt.Errorf("failed to expand inline arguments: %w", err)
	}

	if err := cfg.Init(); err != nil {
		return fmt.Errorf("generating core failed: %w", err)
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.SchemaMutator); ok {
			err := mut.MutateSchema(cfg.Schema)
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name(), err)
			}
		}
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg)
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name(), err)
			}
		}
	}

	// Merge again now that the generated models have been injected into the typemap
	dataPlugins := make([]any, len(plugins))
	for index := range plugins {
		dataPlugins[index] = plugins[index]
	}
	data, err := codegen.BuildData(cfg, dataPlugins...)
	if err != nil {
		return fmt.Errorf("merging type systems failed: %w", err)
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.CodeGenerator); ok {
			err := mut.GenerateCode(data)
			if err != nil {
				return fmt.Errorf("%s: %w", p.Name(), err)
			}
		}
	}

	// Use incremental generation if options provided, otherwise full generation
	if incrementalOpts != nil {
		if err = codegen.GenerateCodeIncremental(data, *incrementalOpts); err != nil {
			return fmt.Errorf("generating core failed: %w", err)
		}
	} else {
		if err = codegen.GenerateCode(data); err != nil {
			return fmt.Errorf("generating core failed: %w", err)
		}
	}

	if !cfg.SkipModTidy {
		if err = cfg.Packages.ModTidy(); err != nil {
			return fmt.Errorf("tidy failed: %w", err)
		}
	}
	if !cfg.SkipValidation {
		if err := validate(cfg); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	return nil
}

func validate(cfg *config.Config) error {
	dirs := []string{cfg.Exec.Dir()}
	if cfg.Model.IsDefined() {
		dirs = append(dirs, cfg.Model.Dir())
	}
	if cfg.Resolver.IsDefined() {
		dirs = append(dirs, cfg.Resolver.Dir())
	}

	// go build runs in the working directory, so the module found there is the main module.
	mainModule := code.ModulePathForDir(".")
	var roots []string
	for _, dir := range dirs {
		dirRoots, err := validationRoots(dir, mainModule)
		if err != nil {
			return err
		}
		roots = append(roots, dirRoots...)
	}

	// Use go build for validation instead of packages.Load with NeedTypes.
	// go build benefits from incremental compilation - only changed files
	// are recompiled. Since we use content-based file writing, unchanged
	// generated files keep their mtime, so go build skips them.
	//
	// FastValidation uses -gcflags="-N -l" to disable compiler
	// optimizations, making cold cache validation ~2x faster.
	return code.ValidateWithBuild(cfg.GetFastValidation(), roots...)
}

// subpackagesWildcard is the Go tooling pattern for "this package and all subpackages".
// Used by go build, go test, etc. (e.g., "go build ./...")
const subpackagesWildcard = "/..."

// validationRoots returns the go build arguments that validate compiles for the output
// directory dir: dir itself and every package beneath it.
//
// Within the main module that is a file path pattern, see buildPattern. The go command refuses
// a file path pattern outside the main module, though, so a directory that belongs to another
// module, such as one wired in with a replace directive or a go.work file, is addressed by its
// import path instead, which resolves through the module graph.
//
// An import path pattern matches nothing when dir lies beneath a directory that the go command
// skips, such as testdata, so the packages there are listed one by one instead. Anywhere else
// the pattern is kept: it only matches packages that the module graph provides, where naming a
// package explicitly fails if, for example, it is missing from the vendor directory.
func validationRoots(dir, mainModule string) ([]string, error) {
	module := code.ModulePathForDir(dir)
	if module == "" || module == mainModule {
		return []string{buildPattern(dir)}, nil
	}
	importPath := code.ImportPathForDir(dir)
	elems := strings.Split(strings.TrimPrefix(importPath, module), "/")
	if !slices.ContainsFunc(elems, skippedByPattern) {
		return []string{importPath + subpackagesWildcard}, nil
	}
	return packageImportPaths(dir)
}

// skippedByPattern reports whether the go command passes over a directory with the given name
// when it expands "...": testdata, and names starting with "." or "_".
func skippedByPattern(name string) bool {
	return name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// packageImportPaths returns the import paths of the packages that the file path pattern
// dir/... matches: dir and every directory beneath it with Go files that build on this
// platform. Like the pattern, it leaves out the directories below dir that skippedByPattern
// reports, and other modules.
func packageImportPaths(dir string) ([]string, error) {
	var importPaths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if path != dir {
			if skippedByPattern(d.Name()) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
		}
		// The pattern passes over a directory without buildable non-test Go files, but naming
		// one makes go build fail. Any other problem is left for go build to report.
		pkg, err := build.ImportDir(path, 0)
		var noGo *build.NoGoError
		if errors.As(err, &noGo) || (err == nil && len(pkg.GoFiles)+len(pkg.CgoFiles) == 0) {
			return nil
		}
		importPaths = append(importPaths, code.ImportPathForDir(path))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing packages in %s: %w", dir, err)
	}
	return importPaths, nil
}

// buildPattern returns a go build pattern matching dir and every package beneath it.
//
// The pattern is expressed as a file path rather than an import path on purpose. When
// expanding "..." in an import path the go command skips directories named testdata, so
// an import path pattern silently matches no packages for output written beneath one and
// validation then succeeds having compiled nothing.
//
// PackageConfig.Dir is normally absolute and slash separated already, because the config
// runs filenames through filepath.Abs and ToSlash when it loads them.
func buildPattern(dir string) string {
	pattern := filepath.ToSlash(dir)
	if pattern == "" {
		pattern = "."
	}
	// "./" is what marks the pattern as a file path; paths that are already relative to
	// the current or parent directory, or rooted, are unambiguous without it.
	if !strings.HasPrefix(pattern, ".") && !isRootedSlashPath(pattern) {
		pattern = "./" + pattern
	}
	return pattern + subpackagesWildcard
}

// isRootedSlashPath reports whether the slash separated path p is absolute.
//
// It recognises Windows drive letters and UNC paths on every OS rather than deferring to
// filepath.IsAbs, which only understands the host's own convention. A Windows path
// therefore has to be classified correctly when the tests run on Linux or macOS,
// otherwise "D:/x" gains a "./" prefix and the go command rejects "./D:/x/...".
func isRootedSlashPath(p string) bool {
	// A POSIX root, and also a UNC share once ToSlash has rewritten the separators.
	if strings.HasPrefix(p, "/") {
		return true
	}
	// A drive letter, such as "D:/a/b".
	if len(p) < 2 || p[1] != ':' {
		return false
	}
	c := p[0]
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}
