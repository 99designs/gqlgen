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
		user: User
	}
	type User {
		id: Int
		fist_name: String
	}
`
	serverFilename := "gen/" + name + "/server/server.go"
	cfg := Config{
		SchemaFilename: SchemaFilenames{"schema.graphql"},
		SchemaStr:      map[string]string{"schema.graphql": schema},
		Exec:           PackageConfig{Filename: "gen/" + name + "/exec.go"},
		Model:          PackageConfig{Filename: "gen/" + name + "/model.go"},
		Resolver:       PackageConfig{Filename: "gen/" + name + "/resolver.go", Type: "Resolver"},
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
