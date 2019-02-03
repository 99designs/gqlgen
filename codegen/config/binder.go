package config

import (
	"fmt"
	"go/types"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// Binder connects graphql types to golang types using static analysis
type Binder struct {
	pkgs  []*packages.Package
	types TypeMap
}

func (c *Config) NewBinder() (*Binder, error) {
	_, runtime, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("No gqlgen runtime information")
	}
	var config = packages.Config{
		Dir:  path.Dir(runtime), // Default dir is current context with gqlgen.Types
		Mode: packages.LoadTypes | packages.LoadSyntax,
	}

	var pkgs []*packages.Package
	for _, p := range c.Models.ReferencedPackages() {
		conf := config
		if !strings.HasPrefix(p, "github.com/99designs/gqlgen") {
			conf.Dir = path.Dir(c.Resolver.Filename)
		}

		pck, err := packages.Load(&conf, c.Models.ReferencedPackages()...)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pck...)
	}

	return &Binder{
		pkgs:  pkgs,
		types: c.Models,
	}, nil
}

func (b *Binder) FindType(pkgName string, typeName string) (types.Type, error) {
	obj, err := b.FindObject(pkgName, typeName)
	if err != nil {
		return nil, err
	}

	if fun, isFunc := obj.(*types.Func); isFunc {
		return fun.Type().(*types.Signature).Params().At(0).Type(), nil
	}
	return obj.Type(), nil
}

func (b *Binder) getPkg(find string) *packages.Package {
	for _, p := range b.pkgs {
		if normalizeVendor(find) == normalizeVendor(p.PkgPath) {
			return p
		}
	}
	return nil
}

func (b *Binder) FindObject(pkgName string, typeName string) (types.Object, error) {
	if pkgName == "" {
		return nil, fmt.Errorf("package cannot be nil")
	}
	fullName := typeName
	if pkgName != "" {
		fullName = pkgName + "." + typeName
	}

	pkg := b.getPkg(pkgName)
	if pkg == nil {
		return nil, errors.Errorf("required package was not loaded: %s", fullName)
	}
	for astNode, def := range pkg.TypesInfo.Defs {
		// only look at defs in the top scope
		if def == nil || def.Parent() == nil || def.Parent() != pkg.Types.Scope() {
			continue
		}

		if astNode.Name == typeName || astNode.Name == "Marshal"+typeName {
			return def, nil
		}
	}

	return nil, errors.Errorf("unable to find type %s\n", fullName)
}

var modsRegex = regexp.MustCompile(`^(\*|\[\])*`)

func normalizeVendor(pkg string) string {
	modifiers := modsRegex.FindAllString(pkg, 1)[0]
	pkg = strings.TrimPrefix(pkg, modifiers)
	parts := strings.Split(pkg, "/vendor/")
	return modifiers + parts[len(parts)-1]
}
