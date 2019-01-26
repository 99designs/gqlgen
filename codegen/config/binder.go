package config

import (
	"fmt"
	"go/types"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

// Binder connects graphql types to golang types using static analysis
type Binder struct {
	program *loader.Program
	types   TypeMap
}

func (c *Config) NewBinder() (*Binder, error) {
	conf := loader.Config{
		AllowErrors: true,
		TypeChecker: types.Config{
			Error: func(e error) {},
		},
	}

	for _, pkg := range c.Models.ReferencedPackages() {
		conf.Import(pkg)
	}

	prog, err := conf.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading program")
	}

	return &Binder{
		program: prog,
		types:   c.Models,
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

func (b *Binder) getPkg(find string) *loader.PackageInfo {
	for n, p := range b.program.Imported {
		if normalizeVendor(find) == normalizeVendor(n) {
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

	for astNode, def := range pkg.Defs {
		// only look at defs in the top scope
		if def == nil || def.Parent() == nil || def.Parent() != pkg.Pkg.Scope() {
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
