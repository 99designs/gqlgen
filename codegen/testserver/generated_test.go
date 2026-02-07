package testserver

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLayouts(t *testing.T) {
	singlefileFSet := token.NewFileSet()
	singlefilePkg := loadPackage(t, "singlefile", singlefileFSet)

	followschemaFSet := token.NewFileSet()
	followschemaPkg := loadPackage(t, "followschema", followschemaFSet)

	singlefileDecls := collectDeclStrings(singlefilePkg.Files)
	followschemaDecls := collectDeclStrings(followschemaPkg.Files)

	// Normalize package names so that both sides can be compared.
	for i, s := range followschemaDecls {
		followschemaDecls[i] = strings.ReplaceAll(s, "followschema", "singlefile")
	}

	require.Equal(t, singlefileDecls, followschemaDecls)
}

func loadPackage(t *testing.T, name string, fset *token.FileSet) *ast.Package {
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

func collectDeclStrings(files map[string]*ast.File) []string {
	printFSet := token.NewFileSet()

	var strs []string
	var buf bytes.Buffer
	for _, f := range files {
		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
				continue
			}
			buf.Reset()
			if err := printer.Fprint(&buf, printFSet, decl); err != nil {
				continue
			}
			strs = append(strs, buf.String())
		}
	}

	sort.Strings(strs)
	return strs
}
