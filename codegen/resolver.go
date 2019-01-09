package codegen

import (
	"log"
	"os"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/codegen/unified"
	"github.com/pkg/errors"
)

type ResolverBuild struct {
	*unified.Schema

	PackageName   string
	ResolverType  string
	ResolverFound bool
}

func GenerateResolver(schema *unified.Schema) error {
	resolverBuild, err := buildResolver(schema)
	if err != nil {
		return errors.Wrap(err, "resolver build failed")
	}
	filename := schema.Config.Resolver.Filename

	if resolverBuild.ResolverFound {
		log.Printf("Skipped resolver: %s.%s already exists\n", schema.Config.Resolver.ImportPath(), schema.Config.Resolver.Type)
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

func buildResolver(s *unified.Schema) (*ResolverBuild, error) {
	def, _ := s.FindGoType(s.Config.Resolver.ImportPath(), s.Config.Resolver.Type)
	resolverFound := def != nil

	return &ResolverBuild{
		Schema:        s,
		PackageName:   s.Config.Resolver.Package,
		ResolverType:  s.Config.Resolver.Type,
		ResolverFound: resolverFound,
	}, nil
}
