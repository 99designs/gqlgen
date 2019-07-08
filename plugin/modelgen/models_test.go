package modelgen

import (
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
)

func TestModelGeneration(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen.yml")
	require.NoError(t, err)
	p := Plugin{}
	require.NoError(t, p.MutateConfig(cfg))

	require.True(t, cfg.Models.UserDefined("MissingTypeNotNull"))
	require.True(t, cfg.Models.UserDefined("MissingTypeNullable"))
	require.True(t, cfg.Models.UserDefined("MissingEnum"))
	require.True(t, cfg.Models.UserDefined("MissingUnion"))
	require.True(t, cfg.Models.UserDefined("MissingInterface"))
	require.True(t, cfg.Models.UserDefined("TypeWithDescription"))
	require.True(t, cfg.Models.UserDefined("EnumWithDescription"))
	require.True(t, cfg.Models.UserDefined("InterfaceWithDescription"))
	require.True(t, cfg.Models.UserDefined("UnionWithDescription"))

	t.Run("no pointer pointers", func(t *testing.T) {
		generated, err := ioutil.ReadFile("./out/generated.go")
		require.NoError(t, err)
		require.NotContains(t, string(generated), "**")
	})

	t.Run("description is generated", func(t *testing.T) {
		node, err := parser.ParseFile(token.NewFileSet(), "./out/generated.go", nil, parser.ParseComments)
		require.NoError(t, err)
		for _, commentGroup := range node.Comments {
			text := commentGroup.Text()
			words := strings.Split(text, " ")
			require.True(t, len(words) > 1, "expected description %q to have more than one word", text)
		}
	})
}
