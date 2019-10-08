package introauth

import (
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

type Config struct {
	Directives config.StringList `yaml:"directives"`
}

type Plugin struct {
	cfg Config
}

func New(cfg Config) *Plugin {
	return &Plugin{cfg: cfg}
}

func (p *Plugin) Name() string {
	return "introspection"
}

func (p *Plugin) MutateSchema(schema *ast.Schema) error {
	p.injectIntrospectionAuth(schema)
	return nil
}

func (p *Plugin) GenerateCode(cfg *codegen.Data) error {
	data := Data{
		Data:       cfg,
		Directives: nil,
	}
	for _, directiveName := range p.cfg.Directives {
		for _, directive := range cfg.Schema.Directives {
			if directive.Name == directiveName {
				data.Directives = append(data.Directives, directive)
			}
		}
	}
	return templates.Render(templates.Options{
		PackageName:     cfg.Config.Exec.Package,
		Filename:        "introspection.go",
		RegionTags:      true,
		GeneratedHeader: true,
		Data:            &data,
		Funcs:           nil,
	})
}

func (p *Plugin) injectIntrospectionAuth(schema *ast.Schema) {
	introspection := &ast.DirectiveDefinition{
		Description: "",
		Name:        "introspection",
		Arguments:   nil,
		Locations: []ast.DirectiveLocation{
			ast.LocationInputFieldDefinition,
			ast.LocationFieldDefinition,
		},
		Position: nil,
	}
	schema.Directives["introspection"] = introspection
	__type := schema.Types["__Type"]
	fields := __type.Fields.ForName("fields")
	fields.Directives = append(fields.Directives, &ast.Directive{
		Name:             introspection.Name,
		Arguments:        nil,
		Position:         nil,
		ParentDefinition: nil,
		Definition:       introspection,
		Location:         ast.LocationFieldDefinition,
	})
	inputFields := __type.Fields.ForName("inputFields")
	inputFields.Directives = append(inputFields.Directives, &ast.Directive{
		Name:             introspection.Name,
		Arguments:        nil,
		Position:         nil,
		ParentDefinition: nil,
		Definition:       introspection,
		Location:         ast.LocationFieldDefinition,
	})
}

type Data struct {
	*codegen.Data
	Directives []*ast.DirectiveDefinition
}
