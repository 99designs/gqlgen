package codegen

import (
	"testing"

	"github.com/vektah/gqlparser/gqlerror"

	"github.com/vektah/gqlparser/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"golang.org/x/tools/go/loader"
)

func TestTypeUnionAsInput(t *testing.T) {
	err := generate("inputunion", `
		type Query {
			addBookmark(b: Bookmarkable!): Boolean!
		}
		type Item {name: String}
		union Bookmarkable = Item
	`)

	require.EqualError(t, err, "model plan failed: Bookmarkable! cannot be used as argument of Query.addBookmark. only input and scalar types are allowed")
}

func TestTypeInInput(t *testing.T) {
	err := generate("typeinput", `
		type Query {
			addBookmark(b: BookmarkableInput!): Boolean!
		}
		type Item {name: String}
		input BookmarkableInput {
			item: Item
		}
	`)

	require.EqualError(t, err, "model plan failed: Item cannot be used as a field of BookmarkableInput. only input and scalar types are allowed")
}

func generate(name string, schema string, typemap ...config.TypeMap) error {
	gen := Generator{
		Config: &config.Config{
			SchemaFilename: config.SchemaFilenames{"schema.graphql"},
			Exec:           config.PackageConfig{Filename: "gen/" + name + "/exec.go"},
			Model:          config.PackageConfig{Filename: "gen/" + name + "/model.go"},
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

	if len(typemap) > 0 {
		gen.Models = typemap[0]
	}
	err = gen.Generate()
	if err != nil {
		return err
	}
	conf := loader.Config{}
	conf.Import("github.com/99designs/gqlgen/codegen/testdata/gen/" + name)

	_, err = conf.Load()
	if err != nil {
		panic(err)
	}
	return nil
}
