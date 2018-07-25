package codegen

import (
	"fmt"
	"go/build"
	"go/types"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

type Build struct {
	PackageName      string
	Objects          Objects
	Inputs           Objects
	Interfaces       []*Interface
	Imports          []*Import
	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	SchemaRaw        string
	SchemaFilename   string
	Directives       []*Directive
}

type ModelBuild struct {
	PackageName string
	Imports     []*Import
	Models      []Model
	Enums       []Enum
}

// Create a list of models that need to be generated
func (cfg *Config) models() (*ModelBuild, error) {
	namedTypes := cfg.buildNamedTypes()

	prog, err := cfg.loadProgram(namedTypes, true)
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}
	imports := buildImports(namedTypes, cfg.Model.Dir())

	cfg.bindTypes(imports, namedTypes, cfg.Model.Dir(), prog)

	models, err := cfg.buildModels(namedTypes, prog)
	if err != nil {
		return nil, err
	}
	return &ModelBuild{
		PackageName: cfg.Model.Package,
		Models:      models,
		Enums:       cfg.buildEnums(namedTypes),
		Imports:     imports.finalize(),
	}, nil
}

// bind a schema together with some code to generate a Build
func (cfg *Config) bind() (*Build, error) {
	namedTypes := cfg.buildNamedTypes()

	prog, err := cfg.loadProgram(namedTypes, true)
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	imports := buildImports(namedTypes, cfg.Exec.Dir())
	cfg.bindTypes(imports, namedTypes, cfg.Exec.Dir(), prog)

	objects, err := cfg.buildObjects(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}

	inputs, err := cfg.buildInputs(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}

	b := &Build{
		PackageName:    cfg.Exec.Package,
		Objects:        objects,
		Interfaces:     cfg.buildInterfaces(namedTypes, prog),
		Inputs:         inputs,
		Imports:        imports.finalize(),
		SchemaRaw:      cfg.SchemaStr,
		SchemaFilename: cfg.SchemaFilename,
		Directives:     cfg.buildDirectives(),
	}

	if cfg.schema.Query != nil {
		b.QueryRoot = b.Objects.ByName(cfg.schema.Query.Name)
	} else {
		return b, fmt.Errorf("query entry point missing")
	}

	if cfg.schema.Mutation != nil {
		b.MutationRoot = b.Objects.ByName(cfg.schema.Mutation.Name)
	}

	if cfg.schema.Subscription != nil {
		b.SubscriptionRoot = b.Objects.ByName(cfg.schema.Subscription.Name)
	}
	return b, nil
}

func (cfg *Config) validate() error {
	namedTypes := cfg.buildNamedTypes()

	_, err := cfg.loadProgram(namedTypes, false)
	return err
}

func (cfg *Config) loadProgram(namedTypes NamedTypes, allowErrors bool) (*loader.Program, error) {
	conf := loader.Config{}
	if allowErrors {
		conf = loader.Config{
			AllowErrors: true,
			TypeChecker: types.Config{
				Error: func(e error) {},
			},
		}
	}
	for _, imp := range ambientImports {
		conf.Import(imp)
	}

	for _, imp := range namedTypes {
		if imp.Package != "" {
			conf.Import(imp.Package)
		}
	}

	return conf.Load()
}

func resolvePkg(pkgName string) (string, error) {
	cwd, _ := os.Getwd()

	pkg, err := build.Default.Import(pkgName, cwd, build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.ImportPath, nil
}
