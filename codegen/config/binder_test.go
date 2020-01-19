package config

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/packages"
)

func TestBindingToInvalid(t *testing.T) {
	binder, schema := createBinder(Config{})
	_, err := binder.TypeReference(schema.Query.Fields.ForName("messages").Type, &types.Basic{})
	require.EqualError(t, err, "Message has an invalid type")
}

func TestSlicePointerBinding(t *testing.T) {
	t.Run("without OmitSliceElementPointers", func(t *testing.T) {
		binder, schema := createBinder(Config{
			OmitSliceElementPointers: false,
		})

		ta, err := binder.TypeReference(schema.Query.Fields.ForName("messages").Type, nil)
		if err != nil {
			panic(err)
		}

		require.Equal(t, ta.GO.String(), "[]*github.com/99designs/gqlgen/example/chat.Message")
	})

	t.Run("with OmitSliceElementPointers", func(t *testing.T) {
		binder, schema := createBinder(Config{
			OmitSliceElementPointers: true,
		})

		ta, err := binder.TypeReference(schema.Query.Fields.ForName("messages").Type, nil)
		if err != nil {
			panic(err)
		}

		require.Equal(t, ta.GO.String(), "[]github.com/99designs/gqlgen/example/chat.Message")
	})
}

func createBinder(cfg Config) (*Binder, *ast.Schema) {
	cfg.Models = TypeMap{
		"Message": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/example/chat.Message"},
		},
	}

	s := gqlparser.MustLoadSchema(&ast.Source{Name: "TestAutobinding.schema", Input: `
		type Message { id: ID }

		type Query {
			messages: [Message!]!
		}
	`})

	ps, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesSizes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
	}, cfg.Models.ReferencedPackages()...)
	if err != nil {
		panic(err)
	}

	b, err := cfg.NewBinder(s, ps)
	if err != nil {
		panic(err)
	}

	return b, s
}
