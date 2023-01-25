package code

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompatibleTypes(t *testing.T) {
	valid := []struct {
		expected string
		actual   string
	}{
		{"string", "string"},
		{"*string", "string"},
		{"string", "*string"},
		{"*string", "*string"},
		{"[]string", "[]string"},
		{"*[]string", "[]string"},
		{"*[]string", "[]*string"},
		{"*[]*[]*[]string", "[][][]string"},
		{"map[string]interface{}", "map[string]interface{}"},
		{"map[string]string", "map[string]string"},
		{"Bar", "Bar"},
		{"interface{}", "interface{}"},
		{"interface{Foo() bool}", "interface{Foo() bool}"},
		{"struct{Foo bool}", "struct{Foo bool}"},
	}

	for _, tc := range valid {
		t.Run(tc.expected+"="+tc.actual, func(t *testing.T) {
			expectedType := parseTypeStr(t, tc.expected)
			actualType := parseTypeStr(t, tc.actual)
			require.NoError(t, CompatibleTypes(expectedType, actualType))
		})
	}

	invalid := []struct {
		expected string
		actual   string
	}{
		{"string", "int"},
		{"*string", "[]string"},
		{"[]string", "[][]string"},
		{"Bar", "Baz"},
		{"map[string]interface{}", "map[string]string"},
		{"map[string]string", "[]string"},
		{"interface{Foo() bool}", "interface{}"},
		{"struct{Foo bool}", "struct{Bar bool}"},
	}

	for _, tc := range invalid {
		t.Run(tc.expected+"!="+tc.actual, func(t *testing.T) {
			expectedType := parseTypeStr(t, tc.expected)
			actualType := parseTypeStr(t, tc.actual)
			require.Error(t, CompatibleTypes(expectedType, actualType))
		})
	}
}

func TestSimilarBasicKind(t *testing.T) {
	valid := []struct {
		name     string
		expected types.BasicKind
		actual   types.BasicKind
	}{
		{name: "string=string", expected: types.String, actual: types.String},
		{name: "int8=int", expected: types.Int64, actual: types.Int8},
		{name: "byte=byte", expected: types.Byte, actual: types.Byte},
	}

	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, similarBasicKind(tc.actual))

		})
	}
}

func parseTypeStr(t *testing.T, s string) types.Type {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", `package test
		type Bar string
		type Baz string

		type Foo struct {
			Field `+s+`
		}
	`, 0)
	require.NoError(t, err)

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("test", fset, []*ast.File{f}, nil)
	require.NoError(t, err)

	return pkg.Scope().Lookup("Foo").Type().(*types.Named).Underlying().(*types.Struct).Field(0).Type()
}
