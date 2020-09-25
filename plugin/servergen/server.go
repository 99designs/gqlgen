package servergen

import (
	"log"
	"os"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
	"github.com/pkg/errors"
)

const testTemplate = `
{{ reserveImport "testing" }}
{{ reserveImport "github.com/99designs/gqlgen/client" }}

func TestIntrospectTodo(t *testing.T) {
	var resp struct {
		Type struct {
			Kind string
		}
	}
	c := client.New(createHandler(&{{ lookupImport .ResolverPackageName}}.Resolver{}))
	c.MustPost("query ($type: String!){ type:__type(name: $type) { kind } }", &resp, client.Var("type", "Todo"))

	if resp.Type.Kind != "OBJECT" {
		t.Error("Unexpected kind for TODO. Expected OBJECT, found " + resp.Type.Kind)
	}
}
`

func New(filename string) plugin.Plugin {
	return &Plugin{filename}
}

type Plugin struct {
	filename string
}

var _ plugin.CodeGenerator = &Plugin{}

func (m *Plugin) Name() string {
	return "servergen"
}
func (m *Plugin) GenerateCode(data *codegen.Data) error {
	serverBuild := &ServerBuild{
		ExecPackageName:     data.Config.Exec.ImportPath(),
		ResolverPackageName: data.Config.Resolver.ImportPath(),
	}

	if _, err := os.Stat(m.filename); os.IsNotExist(errors.Cause(err)) {
		err := templates.Render(templates.Options{
			PackageName: "main",
			Filename:    strings.Replace(m.filename, ".go", "_test.go", 1),
			Data:        serverBuild,
			Packages:    data.Config.Packages,
			Template:    testTemplate,
		})
		if err != nil {
			return err
		}
		return templates.Render(templates.Options{
			PackageName: "main",
			Filename:    m.filename,
			Data:        serverBuild,
			Packages:    data.Config.Packages,
		})
	}

	log.Printf("Skipped server: %s already exists\n", m.filename)
	return nil
}

type ServerBuild struct {
	codegen.Data

	ExecPackageName     string
	ResolverPackageName string
}
