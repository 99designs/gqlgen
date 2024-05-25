package testserver

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	eqgo "github.com/kevinmbeaulieu/eq-go/eq-go"
	"github.com/stretchr/testify/require"
)

func TestLayouts(t *testing.T) {
	singlefileFSet := token.NewFileSet()
	singlefilePkg := loadPackage(t, "singlefile", singlefileFSet)

	followschemaFSet := token.NewFileSet()
	followschemaPkg := loadPackage(t, "followschema", followschemaFSet)

	eq, msg := eqgo.PackagesEquivalent(singlefilePkg, singlefileFSet, followschemaPkg, followschemaFSet, nil)
	if !eq {
		// When msg is too long, require.True(...) omits it entirely.
		// Therefore use fmt.Fprintln to print it manually instead.
		fmt.Fprintln(os.Stderr, msg)
		require.Fail(t, "Packages not equivalent")
	}
}

func loadPackage(t *testing.T, name string, fset *token.FileSet) *ast.Package {
	t.Helper()

	path, err := filepath.Abs(name)
	require.NoError(t, err)
	files, err := os.ReadDir(path)
	require.NoError(t, err)

	pkg := ast.Package{
		Name:  name,
		Files: make(map[string]*ast.File),
	}
	for _, f := range files {
		// Only compare generated files.
		if strings.HasSuffix(f.Name(), ".generated.go") ||
			f.Name() == "generated.go" ||
			f.Name() == "resolver.go" ||
			f.Name() == "stub.go" ||
			f.Name() == "models-gen.go" {
			filename := filepath.Join(path, f.Name())
			src, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
			require.NoError(t, err)
			pkg.Files[filename] = src
		}
	}

	return &pkg
}
