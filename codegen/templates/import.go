package templates

import (
	"fmt"
	"go/build"
	"strconv"

	"github.com/99designs/gqlgen/internal/gopath"
)

type Import struct {
	Name  string
	Path  string
	Alias string
}

type Imports struct {
	imports []*Import
	destDir string
}

func (i *Import) String() string {
	if i.Alias == i.Name {
		return strconv.Quote(i.Path)
	}

	return i.Alias + " " + strconv.Quote(i.Path)
}

func (s *Imports) String() string {
	res := ""
	for i, imp := range s.imports {
		if i != 0 {
			res += "\n"
		}
		res += imp.String()
	}
	return res
}

func (s *Imports) Lookup(path string) string {
	if path == "" {
		return ""
	}

	// if we are referencing our own package we dont need an import
	if gopath.MustDir2Import(s.destDir) == path {
		return ""
	}

	if existing := s.findByPath(path); existing != nil {
		return existing.Alias
	}

	pkg, err := build.Default.Import(path, s.destDir, 0)
	if err != nil {
		panic(err)
	}

	imp := &Import{
		Name: pkg.Name,
		Path: path,
	}
	s.imports = append(s.imports, imp)

	alias := imp.Name
	i := 1
	for s.findByAlias(alias) != nil {
		alias = imp.Name + strconv.Itoa(i)
		i++
		if i > 10 {
			panic(fmt.Errorf("too many collisions, last attempt was %s", alias))
		}
	}
	imp.Alias = alias

	return imp.Alias
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
