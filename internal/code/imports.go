package code

import (
	"errors"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

var nameForPackageCacheLock sync.Mutex
var nameForPackageCache []*packages.Package

var gopaths []string

func init() {
	gopaths = filepath.SplitList(build.Default.GOPATH)
	for i, p := range gopaths {
		gopaths[i] = filepath.ToSlash(filepath.Join(p, "src"))
	}
}

// NameForDir manually looks for package stanzas in files located in the given directory. This can be
// much faster than having to consult go list, because we already know exactly where to look.
func NameForDir(dir string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return SanitizePackageName(filepath.Base(dir))
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return SanitizePackageName(filepath.Base(dir))
	}
	fset := token.NewFileSet()
	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".go") {
			continue
		}

		filename := filepath.Join(dir, file.Name())
		if src, err := parser.ParseFile(fset, filename, nil, parser.PackageClauseOnly); err == nil {
			return src.Name.Name
		}
	}

	return SanitizePackageName(filepath.Base(dir))
}

// goModuleRoot returns the root of the current go module if there is a go.mod file in the directory tree
// If not, it returns false
func goModuleRoot(dir string) (string, bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	dir = filepath.ToSlash(dir)
	modDir := dir
	assumedPart := ""
	for {
		f, err := ioutil.ReadFile(filepath.Join(modDir, "go.mod"))
		if err == nil {
			// found it, stop searching
			return string(modregex.FindSubmatch(f)[1]) + assumedPart, true
		}

		assumedPart = "/" + filepath.Base(modDir) + assumedPart
		parentDir, err := filepath.Abs(filepath.Join(modDir, ".."))
		if err != nil {
			panic(err)
		}

		if parentDir == modDir {
			// Walked all the way to the root and didnt find anything :'(
			break
		}
		modDir = parentDir
	}
	return "", false
}

// ImportPathForDir takes a path and returns a golang import path for the package
func ImportPathForDir(dir string) (res string) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	dir = filepath.ToSlash(dir)

	modDir, ok := goModuleRoot(dir)
	if ok {
		return modDir
	}

	for _, gopath := range gopaths {
		if len(gopath) < len(dir) && strings.EqualFold(gopath, dir[0:len(gopath)]) {
			return dir[len(gopath)+1:]
		}
	}

	return ""
}

var modregex = regexp.MustCompile("module (.*)\n")

// RecordPackagesList records the list of packages to be used later by NameForPackage.
// It must be called exactly once during initialization, before NameForPackage is called.
func RecordPackagesList(newNameForPackageCache []*packages.Package) {
	nameForPackageCache = newNameForPackageCache
}

// NameForPackage returns the package name for a given import path. This can be really slow.
func NameForPackage(importPath string) string {
	if importPath == "" {
		panic(errors.New("import path can not be empty"))
	}
	if nameForPackageCache == nil {
		panic(fmt.Errorf("NameForPackage called for %s before RecordPackagesList", importPath))
	}
	nameForPackageCacheLock.Lock()
	defer nameForPackageCacheLock.Unlock()
	var p *packages.Package
	for _, pkg := range nameForPackageCache {
		if pkg.PkgPath == importPath {
			p = pkg
			break
		}
	}

	if p == nil || p.Name == "" {
		return SanitizePackageName(filepath.Base(importPath))
	}
	return p.Name
}
