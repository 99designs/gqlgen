package codegen

import (
	"fmt"
	"go/build"
	"go/types"
	"os"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

type Build struct {
	PackageName      string
	Objects          Objects
	Inputs           Objects
	Interfaces       []*Interface
	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	SchemaRaw        map[string]string
	SchemaFilename   config.SchemaFilenames
	Directives       map[string]*Directive
}

type ModelBuild struct {
	PackageName string
	Models      []Model
	Enums       []Enum
}

type ResolverBuild struct {
	PackageName   string
	ResolverType  string
	Objects       Objects
	ResolverFound bool
}

type ServerBuild struct {
	PackageName         string
	ExecPackageName     string
	ResolverPackageName string
}

// Create a list of models that need to be generated
func (g *Generator) models() (*ModelBuild, error) {
	progLoader := g.newLoaderWithoutErrors()

	prog, err := progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	namedTypes, err := g.buildNamedTypes(prog)
	if err != nil {
		return nil, errors.Wrap(err, "binding types failed")
	}

	directives, err := g.buildDirectives(namedTypes)
	if err != nil {
		return nil, err
	}
	g.Directives = directives

	models, err := g.buildModels(namedTypes, prog)
	if err != nil {
		return nil, err
	}
	return &ModelBuild{
		PackageName: g.Model.Package,
		Models:      models,
		Enums:       g.buildEnums(namedTypes),
	}, nil
}

// bind a schema together with some code to generate a Build
func (g *Generator) resolver() (*ResolverBuild, error) {
	progLoader := g.newLoaderWithoutErrors()
	progLoader.Import(g.Resolver.ImportPath())

	prog, err := progLoader.Load()
	if err != nil {
		return nil, err
	}

	namedTypes, err := g.buildNamedTypes(prog)
	if err != nil {
		return nil, errors.Wrap(err, "binding types failed")
	}

	directives, err := g.buildDirectives(namedTypes)
	if err != nil {
		return nil, err
	}
	g.Directives = directives

	objects, err := g.buildObjects(namedTypes, prog)
	if err != nil {
		return nil, err
	}

	def, _ := findGoType(prog, g.Resolver.ImportPath(), g.Resolver.Type)
	resolverFound := def != nil

	return &ResolverBuild{
		PackageName:   g.Resolver.Package,
		Objects:       objects,
		ResolverType:  g.Resolver.Type,
		ResolverFound: resolverFound,
	}, nil
}

func (g *Generator) server(destDir string) *ServerBuild {
	return &ServerBuild{
		PackageName:         g.Resolver.Package,
		ExecPackageName:     g.Exec.ImportPath(),
		ResolverPackageName: g.Resolver.ImportPath(),
	}
}

// bind a schema together with some code to generate a Build
func (g *Generator) bind() (*Build, error) {
	progLoader := g.newLoaderWithoutErrors()
	prog, err := progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	namedTypes, err := g.buildNamedTypes(prog)
	if err != nil {
		return nil, errors.Wrap(err, "binding types failed")
	}

	directives, err := g.buildDirectives(namedTypes)
	if err != nil {
		return nil, err
	}
	g.Directives = directives

	objects, err := g.buildObjects(namedTypes, prog)
	if err != nil {
		return nil, err
	}

	inputs, err := g.buildInputs(namedTypes, prog)
	if err != nil {
		return nil, err
	}

	b := &Build{
		PackageName:    g.Exec.Package,
		Objects:        objects,
		Interfaces:     g.buildInterfaces(namedTypes, prog),
		Inputs:         inputs,
		SchemaRaw:      g.SchemaStr,
		SchemaFilename: g.SchemaFilename,
		Directives:     directives,
	}

	if g.schema.Query != nil {
		b.QueryRoot = b.Objects.ByName(g.schema.Query.Name)
	} else {
		return b, fmt.Errorf("query entry point missing")
	}

	if g.schema.Mutation != nil {
		b.MutationRoot = b.Objects.ByName(g.schema.Mutation.Name)
	}

	if g.schema.Subscription != nil {
		b.SubscriptionRoot = b.Objects.ByName(g.schema.Subscription.Name)
	}
	return b, nil
}

func (g *Generator) validate() error {
	progLoader := g.newLoaderWithErrors()
	_, err := progLoader.Load()
	return err
}

func (g *Generator) newLoaderWithErrors() loader.Config {
	conf := loader.Config{}

	for _, pkg := range g.Models.ReferencedPackages() {
		conf.Import(pkg)
	}
	return conf
}

func (g *Generator) newLoaderWithoutErrors() loader.Config {
	conf := g.newLoaderWithErrors()
	conf.AllowErrors = true
	conf.TypeChecker = types.Config{
		Error: func(e error) {},
	}
	return conf
}

func resolvePkg(pkgName string) (string, error) {
	cwd, _ := os.Getwd()

	pkg, err := build.Default.Import(pkgName, cwd, build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.ImportPath, nil
}
