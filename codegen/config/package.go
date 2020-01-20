package config

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/internal/code"
)

type PackageConfig struct {
	Filename string `yaml:"filename,omitempty"`
	Package  string `yaml:"package,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

func (c *PackageConfig) normalize() error {
	if c.Filename != "" {
		c.Filename = abs(c.Filename)
	}
	// If Package is not set, first attempt to load the package at the output dir. If that fails
	// fallback to just the base dir name of the output filename.
	if c.Package == "" {
		c.Package = code.NameForDir(c.Dir())
	}

	return nil
}

func (c *PackageConfig) ImportPath() string {
	return code.ImportPathForDir(c.Dir())
}

func (c *PackageConfig) Dir() string {
	return filepath.Dir(c.Filename)
}

func (c *PackageConfig) Pkg() *types.Package {
	return types.NewPackage(c.ImportPath(), c.Dir())
}

func (c *PackageConfig) IsDefined() bool {
	return c.Filename != ""
}

func (c *PackageConfig) Check(filesMap map[string]bool, pkgConfigsByDir map[string]*PackageConfig) error {
	if err := c.normalize(); err != nil {
		return err
	}
	if strings.ContainsAny(c.Package, "./\\") {
		return fmt.Errorf("package should be the output package name only, do not include the output filename")
	}
	if c.Filename != "" && !strings.HasSuffix(c.Filename, ".go") {
		return fmt.Errorf("filename should be path to a go source file")
	}

	_, fileFound := filesMap[c.Filename]
	if fileFound {
		return fmt.Errorf("filename %s defined more than once", c.Filename)
	}
	filesMap[c.Filename] = true
	previous, inSameDir := pkgConfigsByDir[c.Dir()]
	if inSameDir && c.Package != previous.Package {
		return fmt.Errorf("filenames %s and %s are in the same directory but have different package definitions (%s vs %s)", stripPath(c.Filename), stripPath(previous.Filename), c.Package, previous.Package)
	}
	pkgConfigsByDir[c.Dir()] = c
	return nil
}
