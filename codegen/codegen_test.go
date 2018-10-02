package codegen

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/loader"
)

func TestGenerateServer(t *testing.T) {
	name := "graphserver"
	schema := `
	type Query {
		user(): User
	}
	type User {
		id: Int
	}
`
	serverFilename := "gen/" + name + "/server/server.go"
	cfg := Config{
		SchemaStr: schema,
		Exec:      PackageConfig{Filename: "gen/" + name + "/exec.go"},
		Model:     PackageConfig{Filename: "gen/" + name + "/model.go"},
		Resolver:  PackageConfig{Filename: "gen/" + name + "/resolver.go", Type: "Resolver"},
	}

	_ = syscall.Unlink(cfg.Resolver.Filename)
	_ = syscall.Unlink(serverFilename)

	err := Generate(cfg)
	require.NoError(t, err)

	err = GenerateServer(cfg, serverFilename)
	require.NoError(t, err)

	conf := loader.Config{}
	conf.CreateFromFilenames("gen/"+name, serverFilename)

	_, err = conf.Load()
	require.NoError(t, err)
}

func generate(name string, schema string, typemap ...TypeMap) error {
	cfg := Config{
		SchemaStr: schema,
		Exec:      PackageConfig{Filename: "gen/" + name + "/exec.go"},
		Model:     PackageConfig{Filename: "gen/" + name + "/model.go"},
	}

	if len(typemap) > 0 {
		cfg.Models = typemap[0]
	}
	err := Generate(cfg)
	if err == nil {
		conf := loader.Config{}
		conf.Import("github.com/99designs/gqlgen/codegen/gen/" + name)

		_, err = conf.Load()
		if err != nil {
			panic(err)
		}
	}
	return err
}
