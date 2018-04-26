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
	Imports          Imports
	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	SchemaRaw        string
}

type ModelBuild struct {
	PackageName string
	Imports     Imports
	Models      []Model
	Enums       []Enum
}

// Create a list of models that need to be generated
func (cfg *Config) models() (*ModelBuild, error) {
	namedTypes := cfg.buildNamedTypes()

	imports := buildImports(namedTypes, cfg.modelDir)
	prog, err := cfg.loadProgram(imports, true)
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	cfg.bindTypes(imports, namedTypes, cfg.modelDir, prog)

	models, err := cfg.buildModels(namedTypes, prog)
	if err != nil {
		return nil, err
	}
	return &ModelBuild{
		PackageName: cfg.ModelPackageName,
		Models:      models,
		Enums:       cfg.buildEnums(namedTypes),
		Imports:     buildImports(namedTypes, cfg.modelDir),
	}, nil
}

// bind a schema together with some code to generate a Build
func (cfg *Config) bind() (*Build, error) {
	namedTypes := cfg.buildNamedTypes()

	imports := buildImports(namedTypes, cfg.execDir)
	prog, err := cfg.loadProgram(imports, false)
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	imports = cfg.bindTypes(imports, namedTypes, cfg.execDir, prog)

	objects, err := cfg.buildObjects(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}

	inputs, err := cfg.buildInputs(namedTypes, prog, imports)
	if err != nil {
		return nil, err
	}

	b := &Build{
		PackageName: cfg.ExecPackageName,
		Objects:     objects,
		Interfaces:  cfg.buildInterfaces(namedTypes, prog),
		Inputs:      inputs,
		Imports:     imports,
	}

	if qr, ok := cfg.schema.EntryPoints["query"]; ok {
		b.QueryRoot = b.Objects.ByName(qr.TypeName())
	}

	if mr, ok := cfg.schema.EntryPoints["mutation"]; ok {
		b.MutationRoot = b.Objects.ByName(mr.TypeName())
	}

	if sr, ok := cfg.schema.EntryPoints["subscription"]; ok {
		b.SubscriptionRoot = b.Objects.ByName(sr.TypeName())
	}

	if b.QueryRoot == nil {
		return b, fmt.Errorf("query entry point missing")
	}

	// Poke a few magic methods into query
	q := b.Objects.ByName(b.QueryRoot.GQLType)
	q.Fields = append(q.Fields, Field{
		Type:         &Type{namedTypes["__Schema"], []string{modPtr}, ""},
		GQLName:      "__schema",
		NoErr:        true,
		GoMethodName: "ec.introspectSchema",
		Object:       q,
	})
	q.Fields = append(q.Fields, Field{
		Type:         &Type{namedTypes["__Type"], []string{modPtr}, ""},
		GQLName:      "__type",
		NoErr:        true,
		GoMethodName: "ec.introspectType",
		Args: []FieldArgument{
			{GQLName: "name", Type: &Type{namedTypes["String"], []string{}, ""}, Object: &Object{}},
		},
		Object: q,
	})

	return b, nil
}

func (cfg *Config) loadProgram(imports Imports, allowErrors bool) (*loader.Program, error) {
	conf := loader.Config{}
	if allowErrors {
		conf = loader.Config{
			AllowErrors: true,
			TypeChecker: types.Config{
				Error: func(e error) {},
			},
		}
	}
	for _, imp := range imports {
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
