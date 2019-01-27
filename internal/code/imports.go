package code

import (
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/packages"
)

var pathForDirCache = sync.Map{}

// ImportPathFromDir takes an *absolute* path and returns a golang import path for the package, and returns an error if it isn't on the gopath
func ImportPathForDir(dir string) string {
	if v, ok := pathForDirCache.Load(dir); ok {
		return v.(string)
	}
	p, _ := packages.Load(&packages.Config{
		Dir: dir,
	}, ".")

	if len(p) != 1 {
		return ""
	}

	pathForDirCache.Store(dir, p[0].PkgPath)

	return p[0].PkgPath
}

var nameForPackageCache = sync.Map{}

func NameForPackage(importPath string) string {
	if v, ok := nameForPackageCache.Load(importPath); ok {
		return v.(string)
	}
	p, _ := packages.Load(nil, importPath)

	if len(p) != 1 || p[0].Name == "" {
		return SanitizePackageName(filepath.Base(importPath))
	}

	nameForPackageCache.Store(importPath, p[0].Name)

	return p[0].Name
}
