package config

import (
	"fmt"
	"go/token"
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
		require.NoError(t, err)

		require.Equal(t, "[]*github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message", ta.GO.String())
	})

	t.Run("with OmitSliceElementPointers", func(t *testing.T) {
		binder, schema := createBinder(Config{
			OmitSliceElementPointers: true,
		})

		ta, err := binder.TypeReference(schema.Query.Fields.ForName("messages").Type, nil)
		require.NoError(t, err)

		require.Equal(t, "[]github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat.Message", ta.GO.String())
	})
}

func TestOmittableBinding(t *testing.T) {
	t.Run("bind nullable string with Omittable[string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		require.NoError(t, err)

		it, err := binder.InstantiateType(ot, []types.Type{types.Universe.Lookup("string").Type()})
		require.NoError(t, err)

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableString").Type, it)
		require.NoError(t, err)

		require.True(t, ta.IsOmittable)
	})

	t.Run("bind nullable string with Omittable[*string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		require.NoError(t, err)

		it, err := binder.InstantiateType(ot, []types.Type{types.NewPointer(types.Universe.Lookup("string").Type())})
		require.NoError(t, err)

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableString").Type, it)
		require.NoError(t, err)

		require.True(t, ta.IsOmittable)
	})

	t.Run("fail binding non-nullable string with Omittable[string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		require.NoError(t, err)

		it, err := binder.InstantiateType(ot, []types.Type{types.Universe.Lookup("string").Type()})
		require.NoError(t, err)

		_, err = binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nonNullableString").Type, it)
		require.Error(t, err)
	})

	t.Run("fail binding non-nullable string with Omittable[*string]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		require.NoError(t, err)

		it, err := binder.InstantiateType(ot, []types.Type{types.NewPointer(types.Universe.Lookup("string").Type())})
		require.NoError(t, err)

		_, err = binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nonNullableString").Type, it)
		require.Error(t, err)
	})

	t.Run("bind nullable object with Omittable[T]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		typ, err := binder.FindType("github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat", "Message")
		require.NoError(t, err)

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		require.NoError(t, err)

		it, err := binder.InstantiateType(ot, []types.Type{typ})
		require.NoError(t, err)

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableObject").Type, it)
		require.NoError(t, err)

		require.True(t, ta.IsOmittable)
	})

	t.Run("bind nullable object with Omittable[*T]", func(t *testing.T) {
		binder, schema := createBinder(Config{})

		typ, err := binder.FindType("github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat", "Message")
		require.NoError(t, err)

		ot, err := binder.FindType("github.com/99designs/gqlgen/graphql", "Omittable")
		require.NoError(t, err)

		it, err := binder.InstantiateType(ot, []types.Type{types.NewPointer(typ)})
		require.NoError(t, err)

		ta, err := binder.TypeReference(schema.Types["FooInput"].Fields.ForName("nullableObject").Type, it)
		require.NoError(t, err)

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

func createTypeAlias(name string, t types.Type) *types.Alias {
	var nopos token.Pos
	return types.NewAlias(types.NewTypeName(nopos, nil, name, nil), t)
}

func TestIsNilable(t *testing.T) {
	type aTest struct {
		input    types.Type
		expected bool
	}

	theTests := []aTest{
		{types.Universe.Lookup("any").Type(), true},
		{types.Universe.Lookup("rune").Type(), false},
		{types.Universe.Lookup("byte").Type(), false},
		{types.Universe.Lookup("error").Type(), true},
		{types.Typ[types.Int], false},
		{types.Typ[types.String], false},
		{types.NewChan(types.SendOnly, types.Typ[types.Int]), true},
		{types.NewPointer(types.Typ[types.Int]), true},
		{types.NewPointer(types.Typ[types.String]), true},
		{types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), true},
		{types.NewSlice(types.Typ[types.Int]), true},
		{types.NewInterfaceType(nil, nil), true},
		{createTypeAlias("interfaceAlias", types.NewInterfaceType(nil, nil)), true},
		{createTypeAlias("interfaceNestedAlias", createTypeAlias("interfaceAlias", types.NewInterfaceType(nil, nil))), true},
		{createTypeAlias("intAlias", types.Typ[types.Int]), false},
		{createTypeAlias("intNestedAlias", createTypeAlias("intAlias", types.Typ[types.Int])), false},
	}

	for _, at := range theTests {
		t.Run(fmt.Sprintf("nilable-%s", at.input.String()), func(t *testing.T) {
			require.Equal(t, at.expected, IsNilable(at.input))
		})
	}
}
