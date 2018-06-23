package codegen

import (
	"go/build"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// These imports are referenced by the generated code, and are assumed to have the
// default alias. So lets make sure they get added first, and any later collisions get
// renamed.
var ambientImports = []string{
	"context",
	"fmt",
	"io",
	"strconv",
	"time",
	"sync",
	"github.com/vektah/gqlgen/neelance/introspection",
	"github.com/vektah/gqlgen/neelance/errors",
	"github.com/vektah/gqlgen/neelance/query",
	"github.com/vektah/gqlgen/neelance/schema",
	"github.com/vektah/gqlgen/neelance/validation",
	"github.com/vektah/gqlgen/graphql",
}

func buildImports(types NamedTypes, destDir string) *Imports {
	imports := Imports{
		destDir: destDir,
	}

	for _, ambient := range ambientImports {
		imports.add(ambient)
	}

	// Imports from top level user types
	for _, t := range types {
		t.Import = imports.add(t.Package)
	}

	return &imports
}

var invalidPackageNameChar = regexp.MustCompile(`[^\w]`)

func sanitizePackageName(pkg string) string {
	return invalidPackageNameChar.ReplaceAllLiteralString(filepath.Base(pkg), "_")
}

func (s *Imports) add(path string) *Import {
	if path == "" {
		return nil
	}

	if existing := s.findByPath(path); existing != nil {
		return existing
	}

	pkg, err := build.Default.Import(path, s.destDir, 0)
	if err != nil {
		panic(err)
	}

	alias := ""
	if !strings.HasSuffix(s.destDir, path) {
		if pkg == nil {
			panic(path + " was not loaded")
		}

		alias = pkg.Name
		i := 1
		imp := s.findByAlias(alias)
		for imp != nil && imp.Path != path {
			alias = pkg.Name + strconv.Itoa(i)
			imp = s.findByAlias(alias)
			i++
			if i > 10 {
				panic("too many collisions")
			}
		}
	}

	imp := &Import{
		Alias: alias,
		Path:  path,
	}
	s.imports = append(s.imports, imp)
	sort.Slice(s.imports, func(i, j int) bool {
		return s.imports[i].Alias > s.imports[j].Alias
	})

	return imp
}

func (s Imports) findByPath(importPath string) *Import {
	for _, imp := range s.imports {
		if imp.Path == importPath {
			return imp
		}
	}
	return nil
}

func (s Imports) findByAlias(alias string) *Import {
	for _, imp := range s.imports {
		if imp.Alias == alias {
			return imp
		}
	}
	return nil
}
