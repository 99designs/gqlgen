// +build !go1.9

package main

import "go/importer"

func resolvePkg(pkgName string) (string, error) {
	pkg, err := importer.Default().Import(pkgName)
	if err != nil {
		return "", err
	}
	return pkg.Path(), nil
}
