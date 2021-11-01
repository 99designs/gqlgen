package modelgen

import (
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/modelgen/out"
	"github.com/stretchr/testify/require"
)

func TestModelGeneration(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())
	p := Plugin{
		MutateHook: mutateHook,
		FieldHook:  defaultFieldMutateHook,
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
		file, err := ioutil.ReadFile("./out/generated.go")
		require.NoError(t, err)

		fileText := string(file)

		expectedTags := []string{
			`json:"missing2" database:"MissingTypeNotNullmissing2"`,
			`json:"name" database:"MissingInputname"`,
			`json:"missing2" database:"MissingTypeNullablemissing2"`,
			`json:"name" database:"TypeWithDescriptionname"`,
		}

		for _, tag := range expectedTags {
			require.True(t, strings.Contains(fileText, tag))
		}
	})

	t.Run("field hooks are applied", func(t *testing.T) {
		file, err := ioutil.ReadFile("./out/generated.go")
		require.NoError(t, err)

		fileText := string(file)

		expectedTags := []string{
			`json:"name" anotherTag:"tag"`,
			`json:"enum" yetAnotherTag:"12"`,
			`json:"noVal" yaml:"noVal"`,
			`json:"repeated" someTag:"value" repeated:"true"`,
		}

		for _, tag := range expectedTags {
			require.True(t, strings.Contains(fileText, tag))
		}
	})

	t.Run("concrete types implement interface", func(t *testing.T) {
		var _ out.FooBarer = out.FooBarr{}
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
