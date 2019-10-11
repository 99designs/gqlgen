package modelgen

import (
	"go/ast"
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
	p := Plugin{
		MutateHook: mutateHook,
	}
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

	t.Run("tags are applied", func(t *testing.T) {
		node, err := parser.ParseFile(token.NewFileSet(), "./out/generated.go", nil, 0)
		require.NoError(t, err)
		for _, obj := range node.Scope.Objects {
			if spec, ok := (obj.Decl).(*ast.TypeSpec); ok {
				if st, ok := (spec.Type).(*ast.StructType); ok {
					for _, field := range st.Fields.List {
						fieldName := strings.ToLower(field.Names[0].String())
						expectedTag := "`json:\"" + fieldName + "\" database:\"" + spec.Name.String() + fieldName + "\"`"
						require.True(t, field.Tag.Value == expectedTag)
					}
				}
			}
		}
	})
}

func mutateHook(b *ModelBuild) *ModelBuild {
	for _, model := range b.Models {
		for _, field := range model.Fields {
			field.Tag += ` database:"` + model.Name + field.Name + `"`
		}
	}

	return b
}
