package codegen

import (
	"path/filepath"
	"strconv"
	"strings"
)

func buildImports(types NamedTypes, destDir string) Imports {
	imports := Imports{
		{"context", "context"},
		{"fmt", "fmt"},
		{"io", "io"},
		{"strconv", "strconv"},
		{"time", "time"},
		{"sync", "sync"},
		{"introspection", "github.com/vektah/gqlgen/neelance/introspection"},
		{"errors", "github.com/vektah/gqlgen/neelance/errors"},
		{"query", "github.com/vektah/gqlgen/neelance/query"},
		{"schema", "github.com/vektah/gqlgen/neelance/schema"},
		{"validation", "github.com/vektah/gqlgen/neelance/validation"},
		{"graphql", "github.com/vektah/gqlgen/graphql"},
	}

	for _, t := range types {
		if t.Package == "" {
			continue
		}

		if existing := imports.findByPkg(t.Package); existing != nil {
			t.Import = existing
			continue
		}

		localName := ""
		if !strings.HasSuffix(destDir, t.Package) {
			localName = filepath.Base(t.Package)
			i := 0
			for imp := imports.findByName(localName); imp != nil && imp.Package != t.Package; localName = filepath.Base(t.Package) + strconv.Itoa(i) {
				i++
				if i > 10 {
					panic("too many collisions")
				}
			}
		}

		imp := &Import{
			Name:    localName,
			Package: t.Package,
		}
		t.Import = imp
		imports = append(imports, imp)
	}

	return imports
}

func (i Imports) findByPkg(pkg string) *Import {
	for _, imp := range i {
		if imp.Package == pkg {
			return imp
		}
	}
	return nil
}

func (i Imports) findByName(name string) *Import {
	for _, imp := range i {
		if imp.Name == name {
			return imp
		}
	}
	return nil
}
