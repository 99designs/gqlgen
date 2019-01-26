package gqlgen

import (
	"path/filepath"
	"syscall"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

func Generate(cfg *config.Config, option ...Option) error {
	_ = syscall.Unlink(cfg.Exec.Filename)
	_ = syscall.Unlink(cfg.Model.Filename)

	plugins := []plugin.Plugin{
		modelgen.New(),
	}

	for _, o := range option {
		o(cfg, &plugins)
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
	schema, err := codegen.NewSchema(cfg)
	if err != nil {
		return errors.Wrap(err, "merging failed")
	}

	if err := buildExec(schema); err != nil {
		return errors.Wrap(err, "generating exec failed")
	}

	if cfg.Resolver.IsDefined() {
		if err := GenerateResolver(schema); err != nil {
			return errors.Wrap(err, "generating resolver failed")
		}
	}

	if err := validate(cfg); err != nil {
		return errors.Wrap(err, "validation failed")
	}

	return nil
}

func validate(cfg *config.Config) error {
	conf := loader.Config{}

	conf.Import(cfg.Exec.ImportPath())
	if cfg.Model.IsDefined() {
		conf.Import(cfg.Model.ImportPath())
	}

	if cfg.Resolver.IsDefined() {
		conf.Import(cfg.Resolver.ImportPath())
	}

	_, err := conf.Load()
	return err
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return filepath.ToSlash(absPath)
}
