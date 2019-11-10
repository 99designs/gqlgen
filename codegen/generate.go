package codegen

import (
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/internal/code"
	"golang.org/x/tools/go/packages"
)

func GenerateCode(data *Data, packages []*packages.Package) error {
	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Filename:        data.Config.Exec.Filename,
		Data:            data,
		RegionTags:      true,
		GeneratedHeader: true,
		NameForPackage:  code.NewNameForPackage(packages),
	})
}
