package config

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/internal/code"
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

		require.Equal(t, ta.GO.String(), "[]*github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message")
	})

	t.Run("with OmitSliceElementPointers", func(t *testing.T) {
		binder, schema := createBinder(Config{
			OmitSliceElementPointers: true,
		})

		ta, err := binder.TypeReference(schema.Query.Fields.ForName("messages").Type, nil)
		if err != nil {
			panic(err)
		}

		require.Equal(t, ta.GO.String(), "[]github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message")
	})
}

func TestOmittableBinding(t *testing.T) {
	t.Run("bind nullable string with Omittable[string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		if err != nil {
			panic(err)
		}

		it, err := binder.InstantiateType(ot, []types.Type{types.Universe.Lookup("string").Type()})
		if err != nil {
			panic(err)
		}

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableString").Type, it)
		if err != nil {
			panic(err)
		}

		require.True(t, ta.IsOmittable)
	})

	t.Run("bind nullable string with Omittable[*string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		if err != nil {
			panic(err)
		}

		it, err := binder.InstantiateType(ot, []types.Type{types.NewPointer(types.Universe.Lookup("string").Type())})
		if err != nil {
			panic(err)
		}

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableString").Type, it)
		if err != nil {
			panic(err)
		}

		require.True(t, ta.IsOmittable)
	})

	t.Run("fail binding non-nullable string with Omittable[string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		if err != nil {
			panic(err)
		}

		it, err := binder.InstantiateType(ot, []types.Type{types.Universe.Lookup("string").Type()})
		if err != nil {
			panic(err)
		}

		_, err = binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nonNullableString").Type, it)
		require.Error(t, err)
	})

	t.Run("fail binding non-nullable string with Omittable[*string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		if err != nil {
			panic(err)
		}

		it, err := binder.InstantiateType(ot, []types.Type{types.NewPointer(types.Universe.Lookup("string").Type())})
		if err != nil {
			panic(err)
		}

		_, err = binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nonNullableString").Type, it)
		require.Error(t, err)
	})

	t.Run("bind nullable object with Omittable[T]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		typ, err := binder.FindType("github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat", "Message")
		if err != nil {
			panic(err)
		}

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		if err != nil {
			panic(err)
		}

		it, err := binder.InstantiateType(ot, []types.Type{typ})
		if err != nil {
			panic(err)
		}

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableObject").Type, it)
		if err != nil {
			panic(err)
		}

		require.True(t, ta.IsOmittable)
	})

	t.Run("bind nullable object with Omittable[*T]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		typ, err := binder.FindType("github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat", "Message")
		if err != nil {
			panic(err)
		}

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		if err != nil {
			panic(err)
		}

		it, err := binder.InstantiateType(ot, []types.Type{types.NewPointer(typ)})
		if err != nil {
			panic(err)
		}

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableObject").Type, it)
		if err != nil {
			panic(err)
		}

		require.True(t, ta.IsOmittable)
	})
}

func createBinder(cfg Config) (*Binder, *ast.Schema) {
	cfg.Models = TypeMap{
		"Message": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message"},
		},
		"BarInput": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message"},
		},
		"String": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/graphql.String"},
		},
	}
	cfg.Packages = code.NewPackages()

	cfg.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "TestAutobinding.schema", Input: `
		type Message { id: ID }

		input FooInput {
			nullableString: String
			nonNullableString: String!
			nullableObject: BarInput
		}

		input BarInput {
			id: ID
			text: String!
		}

		type Query {
			messages: [Message!]!
		}
	`})

	b := cfg.NewBinder()

	return b, cfg.Schema
}
