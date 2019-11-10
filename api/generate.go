package api

import (
	"fmt"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/resolvergen"
	"github.com/99designs/gqlgen/plugin/schemaconfig"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

func Generate(cfg *config.Config, option ...Option) error {
	timeStartTotal := time.Now()
	_ = syscall.Unlink(cfg.Exec.Filename)
	_ = syscall.Unlink(cfg.Model.Filename)

	plugins := []plugin.Plugin{
		schemaconfig.New(),
		//modelgen.New(),
		resolvergen.New(),
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

	// load configs now to get packages needed for loading
	schema, schemaStr, err := cfg.LoadSchema()
	if err != nil {
		return err
	}

	err = cfg.Check()
	if err != nil {
		return err
	}

	/// WARNING: now we inject builtins before autobinding because autobinding required package paths to be resolved and injecting builtins can add new paths
	cfg.InjectBuiltins(schema)

	packageNames := append(cfg.AutoBind, cfg.Models.ReferencedPackages()...)
	timeStart := time.Now()
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadTypes | packages.LoadSyntax | packages.NeedName}, packageNames...)
	if err != nil {
		return errors.Wrap(err, "loading failed")
	}
	fmt.Println("loading time ", time.Now().Sub(timeStart))

	// Merge again now that the generated models have been injected into the typemap
	data, err := codegen.BuildData(cfg, pkgs, schema, schemaStr)
	if err != nil {
		return errors.Wrap(err, "merging failed")
	}

	timeStart = time.Now()
	if err = codegen.GenerateCode(data, pkgs); err != nil {
		return errors.Wrap(err, "generating core failed")
	}
	fmt.Println("generation time ", time.Now().Sub(timeStart))

	for _, p := range plugins {
		if mut, ok := p.(plugin.CodeGenerator); ok {
			err := mut.GenerateCode(data)
			if err != nil {
				return errors.Wrap(err, p.Name())
			}
		}
	}

	/*
		timeStart = time.Now()
		if err := validate(cfg); err != nil {
			return errors.Wrap(err, "validation failed")
		}
		fmt.Println("validation time ", time.Now().Sub(timeStart))
	*/
	fmt.Println("total time ", time.Now().Sub(timeStartTotal))

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
