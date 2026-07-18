package api

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"

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

// maskGeneratedOutput hides an existing generated output file from the type
// loader for the duration of this generation run (see the comment at the top
// of generate). The mask is a package-clause-only stub — NOT empty bytes,
// which would be a Go parse error and break loading the rest of the package —
// so the package name is read from the real file's own package clause. A
// missing/unparseable file needs no mask (there is nothing stale to bind to;
// generation will (re)create it), matching how the old unlink was a no-op on
// a missing file.
func maskGeneratedOutput(cfg *config.Config, filename string) {
	if filename == "" {
		return
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return
	}
	if _, err := os.Stat(abs); err != nil {
		return // nothing on disk — nothing stale to mask
	}
	f, err := parser.ParseFile(token.NewFileSet(), abs, nil, parser.PackageClauseOnly)
	if err != nil || f.Name == nil {
		return // can't determine the package — leave it visible rather than corrupt the load
	}
	cfg.MaskGeneratedFile(abs, "package "+f.Name.Name+"\n")
}

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
	// MASK gqlgen's own previous outputs from the type loader, WITHOUT deleting
	// them from disk. If a stale generated model file is visible while the
	// schema loads, autobind finds the previously-generated types in it and
	// binds them as if they were user-written models — so modelgen skips
	// (re)generating them, the freshly-written model file comes out (near-)empty,
	// and the exec build then fails with "unable to find type" (every testserver
	// config that autobinds its own model package hits this). Before this
	// change, api.Generate handled that by syscall.Unlink-ing the outputs up
	// front — but a deleted-then-interrupted generation left the user with NO
	// generated file at all (#2345, #3505). An overlay gives the loader the
	// same "these files don't exist yet" view with no destructive disk write:
	// the real files stay intact until the atomic rename replaces them, and
	// templates.write unmasks each file once its new contents are on disk.
	maskGeneratedOutput(cfg, cfg.Exec.Filename)
	if cfg.Model.IsDefined() {
		maskGeneratedOutput(cfg, cfg.Model.Filename)
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
	roots := []string{withSubpackages(cfg.Exec.ImportPath())}
	if cfg.Model.IsDefined() {
		roots = append(roots, withSubpackages(cfg.Model.ImportPath()))
	}
	if cfg.Resolver.IsDefined() {
		roots = append(roots, withSubpackages(cfg.Resolver.ImportPath()))
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

// withSubpackages appends the Go wildcard pattern to include all subpackages.
func withSubpackages(importPath string) string {
	return importPath + subpackagesWildcard
}
