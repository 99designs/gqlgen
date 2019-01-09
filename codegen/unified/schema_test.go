package unified

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
)

func TestTypeUnionAsInput(t *testing.T) {
	err := generate("inputunion", `testdata/unioninput.graphqls`)

	require.EqualError(t, err, "unable to build object definition: Bookmarkable! cannot be used as argument of Query.addBookmark. only input and scalar types are allowed")
}

func TestTypeInInput(t *testing.T) {
	err := generate("typeinput", `testdata/typeinput.graphqls`)

	require.EqualError(t, err, "unable to build input definition: Item cannot be used as a field of BookmarkableInput. only input and scalar types are allowed")
}

func generate(name string, schemaFilename string) error {
	_, err := NewSchema(&config.Config{
		SchemaFilename: config.SchemaFilenames{schemaFilename},
		Exec:           config.PackageConfig{Filename: "gen/" + name + "/exec.go"},
		Model:          config.PackageConfig{Filename: "gen/" + name + "/model.go"},
	})

	return err
}
