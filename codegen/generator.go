package codegen

import (
	"go/types"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

type Generator struct {
	*config.Config
	schema     *ast.Schema       `yaml:"-"`
	SchemaStr  map[string]string `yaml:"-"`
	Directives map[string]*Directive
}

func New(cfg *config.Config) (*Generator, error) {
	g := &Generator{Config: cfg}

	var err error
	g.schema, g.SchemaStr, err = cfg.LoadSchema()
	if err != nil {
		return nil, err
	}

	err = cfg.Check()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *Generator) Generate() error {
	_ = syscall.Unlink(g.Exec.Filename)
	_ = syscall.Unlink(g.Model.Filename)

	modelsBuild, err := g.models()
	if err != nil {
		return errors.Wrap(err, "model plan failed")
	}
	if len(modelsBuild.Models) > 0 || len(modelsBuild.Enums) > 0 {
		if err = templates.RenderToFile("models.gotpl", g.Model.Filename, modelsBuild); err != nil {
			return err
		}

		for _, model := range modelsBuild.Models {
			modelCfg := g.Models[model.Definition.GQLType]
			modelCfg.Model = types.TypeString(model.Definition.GoType, nil)
			g.Models[model.Definition.GQLType] = modelCfg
		}

		for _, enum := range modelsBuild.Enums {
			modelCfg := g.Models[enum.Definition.GQLType]
			modelCfg.Model = types.TypeString(enum.Definition.GoType, nil)
			g.Models[enum.Definition.GQLType] = modelCfg
		}
	}

	build, err := g.bind()
	if err != nil {
		return errors.Wrap(err, "exec plan failed")
	}

	if err := templates.RenderToFile("generated.gotpl", g.Exec.Filename, build); err != nil {
		return err
	}

	if g.Resolver.IsDefined() {
		if err := g.GenerateResolver(); err != nil {
			return errors.Wrap(err, "generating resolver failed")
		}
	}

	if err := g.validate(); err != nil {
		return errors.Wrap(err, "validation failed")
	}

	return nil
}

func (g *Generator) GenerateServer(filename string) error {
	serverFilename := abs(filename)
	serverBuild := g.server(filepath.Dir(serverFilename))

	if _, err := os.Stat(serverFilename); os.IsNotExist(errors.Cause(err)) {
		err = templates.RenderToFile("server.gotpl", serverFilename, serverBuild)
		if err != nil {
			return errors.Wrap(err, "generate server failed")
		}
	} else {
		log.Printf("Skipped server: %s already exists\n", serverFilename)
	}
	return nil
}

func (g *Generator) GenerateResolver() error {
	resolverBuild, err := g.resolver()
	if err != nil {
		return errors.Wrap(err, "resolver build failed")
	}
	filename := g.Resolver.Filename

	if resolverBuild.ResolverFound {
		log.Printf("Skipped resolver: %s.%s already exists\n", g.Resolver.ImportPath(), g.Resolver.Type)
		return nil
	}

	if _, err := os.Stat(filename); os.IsNotExist(errors.Cause(err)) {
		if err := templates.RenderToFile("resolver.gotpl", filename, resolverBuild); err != nil {
			return err
		}
	} else {
		log.Printf("Skipped resolver: %s already exists\n", filename)
	}

	return nil
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return filepath.ToSlash(absPath)
}
