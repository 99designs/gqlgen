package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"

	"github.com/99designs/gqlgen/internal/code"
)

type typeCacheManifest struct {
	Packages map[string]string `json:"packages"`
}

// LoadTypeCache reads a pre-built type cache directory and populates
// cfg.Packages with the cached data. This allows gqlgen to run without
// a Go module environment (e.g. inside a Bazel sandbox).
//
// The cache directory must contain a manifest.json mapping import paths
// to Go compiler archive files (.a/.x) as produced by rules_go.
func (c *Config) LoadTypeCache(cacheDir string) error {
	manifestPath := filepath.Join(cacheDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading type cache manifest: %w", err)
	}
	var manifest typeCacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing type cache manifest: %w", err)
	}

	pkgs := code.NewPackages(
		code.WithBuildTags(c.GoBuildTags...),
		code.PackagePrefixToCache("github.com/99designs/gqlgen/graphql"),
	)
	fset := token.NewFileSet()
	imports := make(map[string]*types.Package)

	for importPath, filename := range manifest.Packages {
		typesPkg, err := readArchive(filepath.Join(cacheDir, filename), fset, imports, importPath)
		if err != nil {
			return fmt.Errorf("reading export data for %s: %w", importPath, err)
		}

		pkg := &packages.Package{
			ID:        importPath,
			Name:      typesPkg.Name(),
			PkgPath:   importPath,
			Types:     typesPkg,
			TypesInfo: synthesizeTypesInfo(typesPkg),
			Fset:      fset,
		}
		pkgs.Inject(importPath, pkg)
	}

	c.Packages = pkgs
	return nil
}

// readArchive reads Go type information from a Go compiler archive (.a/.x).
func readArchive(path string, fset *token.FileSet, imports map[string]*types.Package, importPath string) (*types.Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := gcexportdata.NewReader(bufio.NewReader(f))
	if err != nil {
		return nil, fmt.Errorf("extracting export data from archive: %w", err)
	}
	return gcexportdata.Read(r, fset, imports, importPath)
}

// synthesizeTypesInfo builds a types.Info with Defs populated from the
// package scope. The binder's indexDefs iterates TypesInfo.Defs to find
// top-level exported names — this provides the same data from export
// data without needing actual source files.
func synthesizeTypesInfo(typesPkg *types.Package) *types.Info {
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}
	scope := typesPkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		ident := ast.NewIdent(name)
		info.Defs[ident] = obj
	}
	return info
}
