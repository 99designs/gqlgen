package codegen

import (
	"testing"

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
		"Shape":      {Model: "github.com/vektah/gqlgen/codegen/testdata.Shape"},
		"ShapeUnion": {Model: "github.com/vektah/gqlgen/codegen/testdata.ShapeUnion"},
		"Circle":     {Model: "github.com/vektah/gqlgen/codegen/testdata.Circle"},
		"Rectangle":  {Model: "github.com/vektah/gqlgen/codegen/testdata.Rectangle"},
	})

	require.NoError(t, err)

}

func generate(name string, schema string, typemap ...TypeMap) error {
	cfg := Config{
		SchemaStr: schema,
		Exec:      PackageConfig{Filename: "testdata/gen/" + name + "/exec.go"},
		Model:     PackageConfig{Filename: "testdata/gen/" + name + "/model.go"},
	}
	if len(typemap) > 0 {
		cfg.Models = typemap[0]
	}
	err := Generate(cfg)
	if err == nil {
		conf := loader.Config{}
		conf.Import("github.com/vektah/gqlgen/codegen/testdata/gen/" + name)

		_, err = conf.Load()
		if err != nil {
			panic(err)
		}
	}
	return err
}
