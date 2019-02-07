package stubgen

import (
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
)

func New(filename string, typename string) plugin.Plugin {
	return &Plugin{filename: filename, typeName: typename}
}

type Plugin struct {
	filename string
	typeName string
}

var _ plugin.CodeGenerator = &Plugin{}

func (m *Plugin) Name() string {
	return "stubgen"
}
func (m *Plugin) GenerateCode(data *codegen.Data) error {
	return templates.Render(templates.Options{
		PackageName: data.Config.Resolver.Package,
		Filename:    m.filename,
		Data: &ResolverBuild{
			Data:     data,
			TypeName: m.typeName,
		},
		GeneratedHeader: false,
	})
}

type ResolverBuild struct {
	*codegen.Data

	TypeName string
}
