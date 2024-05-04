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

		require.Equal(t, "[]*github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message", ta.GO.String())
	})

	t.Run("with OmitSliceElementPointers", func(t *testing.T) {
		binder, schema := createBinder(Config{
			OmitSliceElementPointers: true,
		})

		ta, err := binder.TypeReference(schema.Query.Fields.ForName("messages").Type, nil)
		if err != nil {
			panic(err)
		}

		require.Equal(t, "[]github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message", ta.GO.String())
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

func TestEnumBinding(t *testing.T) {
	cf := Config{}
	cf.Packages = code.NewPackages()
	cf.Models = TypeMap{
		"Bar": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/codegen/config/testdata/enum.Bar"},
			EnumValues: map[string]EnumValue{
				"ONE": {Value: "github.com/99designs/gqlgen/codegen/config/testdata/enum.BarOne"},
				"TWO": {Value: "github.com/99designs/gqlgen/codegen/config/testdata/enum.BarTwo"},
			},
		},
		"Baz": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/graphql.Int"},
			EnumValues: map[string]EnumValue{
				"ONE": {Value: "github.com/99designs/gqlgen/codegen/config/testdata/enum.BazOne"},
				"TWO": {Value: "github.com/99designs/gqlgen/codegen/config/testdata/enum.BazTwo"},
			},
		},
		"String": TypeMapEntry{
			Model: []string{"github.com/99designs/gqlgen/graphql.String"},
		},
	}
	cf.Schema = gqlparser.MustLoadSchema(&ast.Source{Name: "schema", Input: `
	type Query {
	    foo(arg: Bar!): Baz
	}
	
	enum Bar {
	    ONE
	    TWO
	}
	enum Baz {
	    ONE
	    TWO
	}
	`})

	binder := cf.NewBinder()

	barType, err := binder.FindType("github.com/99designs/gqlgen/codegen/config/testdata/enum", "Bar")

	require.NotNil(t, barType)
	require.NoError(t, err)

	bar, err := binder.TypeReference(cf.Schema.Query.Fields.ForName("foo").Arguments.ForName("arg").Type, nil)

	require.NotNil(t, bar)
	require.NoError(t, err)
	require.True(t, bar.HasEnumValues())
	require.Len(t, bar.EnumValues, 2)

	barOne, err := binder.FindObject("github.com/99designs/gqlgen/codegen/config/testdata/enum", "BarOne")

	require.NotNil(t, barOne)
	require.NoError(t, err)
	require.Equal(t, barOne, bar.EnumValues[0].Object)
	require.Equal(t, cf.Schema.Types["Bar"].EnumValues[0], bar.EnumValues[0].Definition)

	barTwo, err := binder.FindObject("github.com/99designs/gqlgen/codegen/config/testdata/enum", "BarTwo")

	require.NotNil(t, barTwo)
	require.NoError(t, err)
	require.Equal(t, barTwo, bar.EnumValues[1].Object)
	require.Equal(t, cf.Schema.Types["Bar"].EnumValues[1], bar.EnumValues[1].Definition)

	bazType, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Int")

	require.NotNil(t, bazType)
	require.NoError(t, err)

	baz, err := binder.TypeReference(cf.Schema.Query.Fields.ForName("foo").Type, nil)

	require.NotNil(t, baz)
	require.NoError(t, err)
	require.True(t, baz.HasEnumValues())
	require.Len(t, baz.EnumValues, 2)

	bazOne, err := binder.FindObject("github.com/99designs/gqlgen/codegen/config/testdata/enum", "BazOne")

	require.NotNil(t, bazOne)
	require.NoError(t, err)
	require.Equal(t, bazOne, baz.EnumValues[0].Object)
	require.Equal(t, cf.Schema.Types["Baz"].EnumValues[0], baz.EnumValues[0].Definition)

	bazTwo, err := binder.FindObject("github.com/99designs/gqlgen/codegen/config/testdata/enum", "BazTwo")

	require.NotNil(t, bazTwo)
	require.NoError(t, err)
	require.Equal(t, bazTwo, baz.EnumValues[1].Object)
	require.Equal(t, cf.Schema.Types["Baz"].EnumValues[1], baz.EnumValues[1].Definition)
}
