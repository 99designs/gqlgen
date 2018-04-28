package codegen

import (
	"path/filepath"
	"regexp"
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
		imports, t.Import = imports.addPkg(types, destDir, t.Package)
	}

	return imports
}

var invalidPackageNameChar = regexp.MustCompile(`[^\w]`)

func sanitizePackageName(pkg string) string {
	return invalidPackageNameChar.ReplaceAllLiteralString(filepath.Base(pkg), "_")
}

func (s Imports) addPkg(types NamedTypes, destDir string, pkg string) (Imports, *Import) {
	if pkg == "" {
		return s, nil
	}

	if existing := s.findByPkg(pkg); existing != nil {
		return s, existing
	}

	localName := ""
	if !strings.HasSuffix(destDir, pkg) {
		localName = sanitizePackageName(filepath.Base(pkg))
		i := 1
		imp := s.findByName(localName)
		for imp != nil && imp.Package != pkg {
			localName = sanitizePackageName(filepath.Base(pkg)) + strconv.Itoa(i)
			imp = s.findByName(localName)
			i++
			if i > 10 {
				panic("too many collisions")
			}
		}
	}

	imp := &Import{
		Name:    localName,
		Package: pkg,
	}
	s = append(s, imp)
	return s, imp
}

func (s Imports) findByPkg(pkg string) *Import {
	for _, imp := range s {
		if imp.Package == pkg {
			return imp
		}
	}
	return nil
}

func (s Imports) findByName(name string) *Import {
	for _, imp := range s {
		if imp.Name == name {
			return imp
		}
	}
	return nil
}
