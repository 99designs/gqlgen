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

	PackageName  string
	ResolverType string
}

func GenerateResolver(schema *unified.Schema) error {
	resolverBuild, err := buildResolver(schema)
	if err != nil {
		return errors.Wrap(err, "resolver build failed")
	}
	filename := schema.Config.Resolver.Filename

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
	return &ResolverBuild{
		Schema:       s,
		PackageName:  s.Config.Resolver.Package,
		ResolverType: s.Config.Resolver.Type,
	}, nil
}
