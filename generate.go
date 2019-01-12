package gqlgen

import (
	"path/filepath"
	"syscall"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
)

func Generate(cfg *config.Config) error {
	_ = syscall.Unlink(cfg.Exec.Filename)
	_ = syscall.Unlink(cfg.Model.Filename)

	schema, err := codegen.NewSchema(cfg)
	if err != nil {
		return errors.Wrap(err, "merging failed")
	}

	if err = buildModels(schema); err != nil {
		return errors.Wrap(err, "generating models failed")
	}

	// Merge again now that the generated models have been injected into the typemap
	schema, err = codegen.NewSchema(schema.Config)
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
	progLoader := cfg.NewLoaderWithErrors()
	_, err := progLoader.Load()
	return err
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return filepath.ToSlash(absPath)
}
