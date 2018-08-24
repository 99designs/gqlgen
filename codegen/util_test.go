package codegen

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeVendor(t *testing.T) {
	require.Equal(t, "bar/baz", normalizeVendor("foo/vendor/bar/baz"))
	require.Equal(t, "[]bar/baz", normalizeVendor("[]foo/vendor/bar/baz"))
	require.Equal(t, "*bar/baz", normalizeVendor("*foo/vendor/bar/baz"))
	require.Equal(t, "*[]*bar/baz", normalizeVendor("*[]*foo/vendor/bar/baz"))
}

func TestFindField(t *testing.T) {
	input := `
package test

type Std struct {
	Name string
	Value int
}
type Anon struct {
	Name string
	Tags
}
type Tags struct {
	Bar string ` + "`" + `gqlgen:"foo"` + "`" + `
	Foo int    ` + "`" + `gqlgen:"bar"` + "`" + `
}
type Amb struct {
	Bar string ` + "`" + `gqlgen:"foo"` + "`" + `
	Foo int    ` + "`" + `gqlgen:"foo"` + "`" + `
}
type Embed struct {
	Std
	Test string
}
`
	scope, err := parseScope(input, "test")
	require.NoError(t, err)

	std := scope.Lookup("Std").Type().Underlying().(*types.Struct)
	anon := scope.Lookup("Anon").Type().Underlying().(*types.Struct)
	tags := scope.Lookup("Tags").Type().Underlying().(*types.Struct)
	amb := scope.Lookup("Amb").Type().Underlying().(*types.Struct)
	embed := scope.Lookup("Embed").Type().Underlying().(*types.Struct)

	tests := []struct {
		Name        string
		Struct      *types.Struct
		Field       string
		Tag         string
		Expected    string
		ShouldError bool
	}{
		{"Finds a field by name with no tag", std, "name", "", "Name", false},
		{"Finds a field by name when passed tag but tag not used", std, "name", "gqlgen", "Name", false},
		{"Ignores tags when not passed a tag", tags, "foo", "", "Foo", false},
		{"Picks field with tag over field name when passed a tag", tags, "foo", "gqlgen", "Bar", false},
		{"Errors when ambigious", amb, "foo", "gqlgen", "", true},
		{"Finds a field that is in embedded struct", anon, "bar", "", "Bar", false},
		{"Finds field that is not in embedded struct", embed, "test", "", "Test", false},
	}

	for _, tt := range tests {
		tt := tt
		field, err := findField(tt.Struct, tt.Field, tt.Tag)
		if tt.ShouldError {
			require.Nil(t, field, tt.Name)
			require.Error(t, err, tt.Name)
		} else {
			require.NoError(t, err, tt.Name)
			require.Equal(t, tt.Expected, field.Name(), tt.Name)
		}
	}
}

func parseScope(input interface{}, packageName string) (*types.Scope, error) {
	// test setup to parse the types
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", input, 0)
	if err != nil {
		return nil, err
	}

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(packageName, fset, []*ast.File{f}, nil)
	if err != nil {
		return nil, err
	}

	return pkg.Scope(), nil
}
