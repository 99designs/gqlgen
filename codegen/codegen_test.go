package codegen

import (
	"syscall"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
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
	enum Status {
		OK
		ERROR
	}
`
	serverFilename := "gen/" + name + "/server/server.go"
	gen := Generator{
		Config: &config.Config{
			SchemaFilename: config.SchemaFilenames{"schema.graphql"},
			Exec:           config.PackageConfig{Filename: "gen/" + name + "/exec.go"},
			Model:          config.PackageConfig{Filename: "gen/" + name + "/model.go"},
			Resolver:       config.PackageConfig{Filename: "gen/" + name + "/resolver.go", Type: "Resolver"},
		},

		SchemaStr: map[string]string{"schema.graphql": schema},
	}

	err := gen.Config.Check()
	if err != nil {
		panic(err)
	}

	var gerr *gqlerror.Error
	gen.schema, gerr = gqlparser.LoadSchema(&ast.Source{Name: "schema.graphql", Input: schema})
	if gerr != nil {
		panic(gerr)
	}

	_ = syscall.Unlink(gen.Resolver.Filename)
	_ = syscall.Unlink(serverFilename)

	err = gen.Generate()
	require.NoError(t, err)

	err = gen.GenerateServer(serverFilename)
	require.NoError(t, err)

	conf := loader.Config{}
	conf.CreateFromFilenames("gen/"+name, serverFilename)

	_, err = conf.Load()
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
