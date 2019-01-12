package gqlgen

import (
	"log"
	"os"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
)

type ServerBuild struct {
	codegen.Schema

	PackageName         string
	ExecPackageName     string
	ResolverPackageName string
}

func GenerateServer(filename string, cfg *config.Config) error {
	serverBuild := buildServer(cfg)

	serverFilename := abs(filename)
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

func buildServer(config *config.Config) *ServerBuild {
	return &ServerBuild{
		PackageName:         config.Resolver.Package,
		ExecPackageName:     config.Exec.ImportPath(),
		ResolverPackageName: config.Resolver.ImportPath(),
	}
}
