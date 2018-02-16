package codegen

import (
	"fmt"
	"go/types"
	"os"

	"golang.org/x/tools/go/loader"
)

func findGoType(prog *loader.Program, pkgName string, typeName string) types.Object {
	fullName := typeName
	if pkgName != "" {
		fullName = pkgName + "." + typeName
	}

	pkgName, err := resolvePkg(pkgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to resolve package for %s: %s\n", fullName, err.Error())
		return nil
	}

	pkg := prog.Imported[pkgName]
	if pkg == nil {
		fmt.Fprintf(os.Stderr, "required package was not loaded: %s", fullName)
		return nil
	}

	for astNode, def := range pkg.Defs {
		if astNode.Name != typeName || isMethod(def) {
			continue
		}

		return def
	}
	fmt.Fprintf(os.Stderr, "unable to find type %s\n", fullName)
	return nil
}

func isMethod(t types.Object) bool {
	f, isFunc := t.(*types.Func)
	if !isFunc {
		return false
	}

	return f.Type().(*types.Signature).Recv() != nil
}
