package api

import (
	"syscall"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/99designs/gqlgen/plugin/resolvergen"
	"github.com/99designs/gqlgen/plugin/schemaconfig"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

func Generate(cfg *config.Config, option ...Option) error {
	_ = syscall.Unlink(cfg.Exec.Filename)
	if cfg.Model.IsDefined() {
		_ = syscall.Unlink(cfg.Model.Filename)
	}
	if err := cfg.Check(); err != nil {
		return errors.Wrap(err, "generating core failed")
	}

	plugins := []plugin.Plugin{schemaconfig.New()}
	if cfg.Model.IsDefined() {
		plugins = append(plugins, modelgen.New())
	}
	plugins = append(plugins, resolvergen.New())
	if cfg.Federated {
		plugins = append([]plugin.Plugin{federation.New()}, plugins...)
	}

	for _, o := range option {
		o(cfg, &plugins)
	}

	schemaMutators := []codegen.SchemaMutator{}
	for _, p := range plugins {
		if inj, ok := p.(plugin.SourcesInjector); ok {
			inj.InjectSources(cfg)
		}
		if mut, ok := p.(codegen.SchemaMutator); ok {
			schemaMutators = append(schemaMutators, mut)
		}
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			err := mut.MutateConfig(cfg)
			if err != nil {
				return errors.Wrap(err, p.Name())
			}
		}
	}
	// Merge again now that the generated models have been injected into the typemap
	data, err := codegen.BuildData(cfg, schemaMutators)
	if err != nil {
		return errors.Wrap(err, "merging type systems failed")
	}

	if err = codegen.GenerateCode(data); err != nil {
		return errors.Wrap(err, "generating code failed")
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.CodeGenerator); ok {
			err := mut.GenerateCode(data)
			if err != nil {
				return errors.Wrap(err, p.Name())
			}
		}
	}

	if err = codegen.GenerateCode(data); err != nil {
		return errors.Wrap(err, "generating core failed")
	}

	if !cfg.SkipValidation {
		if err := validate(cfg); err != nil {
			return errors.Wrap(err, "validation failed")
		}
	}

	return nil
}

func validate(cfg *config.Config) error {
	roots := []string{cfg.Exec.ImportPath()}
	if cfg.Model.IsDefined() {
		roots = append(roots, cfg.Model.ImportPath())
	}

	if cfg.Resolver.IsDefined() {
		roots = append(roots, cfg.Resolver.ImportPath())
	}
	_, err := packages.Load(&packages.Config{Mode: packages.LoadTypes | packages.LoadSyntax}, roots...)
	if err != nil {
		return errors.Wrap(err, "validation failed")
	}
	return nil
}
