package gqlgen

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/loader"
)

func TestGenerateServer(t *testing.T) {
	name := "graphserver"

	cfg := &config.Config{
		SchemaFilename: config.SchemaFilenames{"testdata/generateserver.graphqls"},
		Exec:           config.PackageConfig{Filename: "gen/" + name + "/exec.go"},
		Model:          config.PackageConfig{Filename: "gen/" + name + "/model.go"},
		Resolver:       config.PackageConfig{Filename: "gen/" + name + "/resolver.go", Type: "Resolver"},
	}
	serverFilename := "gen/" + name + "/server/server.go"

	require.NoError(t, Generate(cfg))
	require.NoError(t, GenerateServer(serverFilename, cfg))

	conf := loader.Config{}
	conf.CreateFromFilenames("gen/"+name, serverFilename)

	_, err := conf.Load()
	require.NoError(t, err)

	t.Run("list of enums", func(t *testing.T) {
		conf = loader.Config{}
		conf.CreateFromFilenames("gen/"+name, "gen/"+name+"/model.go")

		program, err := conf.Load()
		require.NoError(t, err)

		found := false

		for _, c := range program.Created {
			for ident := range c.Defs {
				if ident.Name == "AllStatus" {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			t.Fail()
		}
	})
}
