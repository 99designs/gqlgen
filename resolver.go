package gqlgen

import (
	"log"
	"os"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
)

type ResolverBuild struct {
	*codegen.Schema

	PackageName  string
	ResolverType string
}

func GenerateResolver(schema *codegen.Schema) error {
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

func buildResolver(s *codegen.Schema) (*ResolverBuild, error) {
	return &ResolverBuild{
		Schema:       s,
		PackageName:  s.Config.Resolver.Package,
		ResolverType: s.Config.Resolver.Type,
	}, nil
}
