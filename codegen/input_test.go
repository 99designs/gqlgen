package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
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

func generate(name string, schema string, typemap ...TypeMap) error {
	cfg := Config{
		SchemaFilename: SchemaFilenames{"schema.graphql"},
		SchemaStr:      map[string]string{"schema.graphql": schema},
		Exec:           PackageConfig{Filename: "gen/" + name + "/exec.go"},
		Model:          PackageConfig{Filename: "gen/" + name + "/model.go"},
	}

	if len(typemap) > 0 {
		cfg.Models = typemap[0]
	}
	err := Generate(cfg)
	if err == nil {
		conf := loader.Config{}
		conf.Import("github.com/99designs/gqlgen/codegen/testdata/gen/" + name)

		_, err = conf.Load()
		if err != nil {
			panic(err)
		}
	}
	return err
}
