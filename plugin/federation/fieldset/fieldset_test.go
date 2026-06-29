package fieldset

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
)

func TestUnnestedWithoutPrefix(t *testing.T) {
	fieldSet := New("foo bar", nil)

	require.Len(t, fieldSet, 2)

	require.Len(t, fieldSet[0], 1)
	require.Equal(t, "foo", fieldSet[0][0])

	require.Len(t, fieldSet[1], 1)
	require.Equal(t, "bar", fieldSet[1][0])
}

func TestNestedWithoutPrefix(t *testing.T) {
	fieldSet := New("foo bar { baz} a b {c{d}}e", nil)

	require.Len(t, fieldSet, 5)

	require.Len(t, fieldSet[0], 1)
	require.Equal(t, "foo", fieldSet[0][0])

	require.Len(t, fieldSet[1], 2)
	require.Equal(t, "bar", fieldSet[1][0])
	require.Equal(t, "baz", fieldSet[1][1])

	require.Len(t, fieldSet[2], 1)
	require.Equal(t, "a", fieldSet[2][0])

	require.Len(t, fieldSet[3], 3)
	require.Equal(t, "b", fieldSet[3][0])
	require.Equal(t, "c", fieldSet[3][1])
	require.Equal(t, "d", fieldSet[3][2])

	require.Len(t, fieldSet[4], 1)
	require.Equal(t, "e", fieldSet[4][0])
}

func TestWithPrefix(t *testing.T) {
	t.Run("prefix with len=capacity", func(t *testing.T) {
		fieldSet := New("foo bar{id}", []string{"prefix"})

		require.Len(t, fieldSet, 2)

		require.Len(t, fieldSet[0], 2)
		require.Equal(t, "prefix", fieldSet[0][0])
		require.Equal(t, "foo", fieldSet[0][1])

		require.Len(t, fieldSet[1], 3)
		require.Equal(t, "prefix", fieldSet[1][0])
		require.Equal(t, "bar", fieldSet[1][1])
		require.Equal(t, "id", fieldSet[1][2])
	})
	t.Run("prefix with len<capacity", func(t *testing.T) {
		prefix := make([]string, 0, 2)
		prefix = append(prefix, "prefix")
		fieldSet := New("foo bar{id}", prefix)

		require.Len(t, fieldSet, 2)
		t.Log(fieldSet)

		require.Len(t, fieldSet[0], 2)
		require.Equal(t, "prefix", fieldSet[0][0])
		require.Equal(t, "foo", fieldSet[0][1])

		require.Len(t, fieldSet[1], 3)
		require.Equal(t, "prefix", fieldSet[1][0])
		require.Equal(t, "bar", fieldSet[1][1])
		require.Equal(t, "id", fieldSet[1][2])
	})
}

func TestHandlesRequiresFieldWithArgument(t *testing.T) {
	obj := &codegen.Object{
		Fields: []*codegen.Field{
			{
				FieldDefinition: &ast.FieldDefinition{
					Name: "foo(limit:4) { bar }",
				},
				TypeReference:    nil,
				GoFieldType:      0,
				GoReceiverName:   "",
				GoFieldName:      "",
				IsResolver:       false,
				Args:             nil,
				MethodHasContext: false,
				NoErr:            false,
				VOkFunc:          false,
				Object:           nil,
				Default:          nil,
				Stream:           false,
				Directives:       nil,
			},
		},
		Implements: nil,
	}

	require.NotNil(t, fieldByName(obj, "foo"))
}

func TestInvalid(t *testing.T) {
	require.Panics(t, func() {
		New("foo bar{baz", nil)
	})
}

func TestToGo(t *testing.T) {
	require.Equal(t, "Foo", Field{"foo"}.ToGo())
	require.Equal(t, "FooBar", Field{"foo", "bar"}.ToGo())
	require.Equal(t, "BarID", Field{"bar", "id"}.ToGo())
}

func TestToGoPrivate(t *testing.T) {
	require.Equal(t, "foo", Field{"foo"}.ToGoPrivate())
	require.Equal(t, "fooBar", Field{"foo", "bar"}.ToGoPrivate())
	require.Equal(t, "barID", Field{"bar", "id"}.ToGoPrivate())
}

func TestFieldDefinitionMissingField(t *testing.T) {
	inner := &ast.Definition{
		Kind:   ast.Object,
		Name:   "I",
		Fields: ast.FieldList{{Name: "d", Type: ast.NamedType("ID", nil)}},
	}
	testEnt := &ast.Definition{
		Kind: ast.Object,
		Name: "TestEnt",
		Fields: ast.FieldList{
			{Name: "id", Type: ast.NamedType("ID", nil)},
			{Name: "i", Type: ast.NamedType("I", nil)},
		},
	}
	schema := &ast.Schema{Types: map[string]*ast.Definition{"TestEnt": testEnt, "I": inner}}

	// A missing segment must return nil (the "not found" contract) rather than
	// panicking with a nil pointer dereference. The caller turns nil into a
	// clear validation error.
	cases := map[string]Field{
		"missing first segment":        {"order", "id"},  // "order" is not on TestEnt
		"missing intermediate segment": {"i", "missing"}, // "i" exists; "missing" is not on I
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			require.Nil(t, f.FieldDefinition(testEnt, schema))
		})
	}

	// A valid nested path still resolves.
	require.NotNil(t, Field{"i", "d"}.FieldDefinition(testEnt, schema))
}
