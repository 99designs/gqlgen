package codegen

import (
	"testing"

	"syscall"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/loader"
)

func TestShapes(t *testing.T) {
	err := generate("shapes", `
			type Query {
				shapes: [Shape]
			}
			interface Shape {
				area: Float
			}
			type Circle implements Shape {
				radius: Float
				area: Float
			}
			type Rectangle implements Shape {
				length: Float
				width: Float
				area: Float
			}
			union ShapeUnion = Circle | Rectangle
	`, TypeMap{
		"Shape":      {Model: "github.com/99designs/gqlgen/codegen/tests.Shape"},
		"ShapeUnion": {Model: "github.com/99designs/gqlgen/codegen/tests.ShapeUnion"},
		"Circle":     {Model: "github.com/99designs/gqlgen/codegen/tests.Circle"},
		"Rectangle":  {Model: "github.com/99designs/gqlgen/codegen/tests.Rectangle"},
	})

	require.NoError(t, err)

}

func generate(name string, schema string, typemap ...TypeMap) error {
	cfg := Config{
		SchemaStr: schema,
		Exec:      PackageConfig{Filename: "tests/gen/" + name + "/exec.go"},
		Model:     PackageConfig{Filename: "tests/gen/" + name + "/model.go"},
		Resolver:  PackageConfig{Filename: "tests/gen/" + name + "/resolver.go", Type: "Resolver"},
	}

	_ = syscall.Unlink(cfg.Resolver.Filename)

	if len(typemap) > 0 {
		cfg.Models = typemap[0]
	}
	err := Generate(cfg)
	if err == nil {
		conf := loader.Config{}
		conf.Import("github.com/99designs/gqlgen/codegen/tests/gen/" + name)

		_, err = conf.Load()
		if err != nil {
			panic(err)
		}
	}
	return err
}
