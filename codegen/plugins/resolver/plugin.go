package resolver

import (
	"context"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/graphql"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

type Plugin struct{}

var plugin = &Plugin{}

func (p *Plugin) Schema(cfg *codegen.Config) (ast.Source, error) {
	return ast.Source{
		Name:  "resolver plugin",
		Input: "directive @resolver on FIELD_DEFINITION",
	}, nil
}

func (p *Plugin) Execute(cfg *codegen.Config, schema *ast.Schema) error {
	if cfg.Directives.ImplementationFor("resolver") != "" {
		return errors.Errorf("directive implementation for resolver already exists")
	}
	cfg.Directives["resolver"] = codegen.DirectiveMapEntry{Implementation: "github.com/99designs/gqlgen/codegen/plugins/resolver.DirectiveNoop"}
	for _, typ := range schema.Types {
		for _, f := range typ.Fields {
			for _, d := range f.Directives {
				if d.Name == "resolver" {
					if _, ok := cfg.Models[typ.Name]; !ok {
						cfg.Models[typ.Name] = codegen.TypeMapEntry{}
					}
					if tmf, ok := cfg.Models[typ.Name].Fields[f.Name]; ok {
						cfg.Models[typ.Name].Fields[f.Name] = codegen.TypeMapField{
							ForceResolver: true,
							FieldName:     tmf.FieldName,
						}
					} else {
						cfg.Models[typ.Name].Fields[f.Name] = codegen.TypeMapField{
							ForceResolver: true,
						}
					}
				}
			}
		}
	}
	return nil
}

func DirectiveNoop(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	return next(ctx)
}

func init() {
	codegen.RegisterPlugin(plugin)
}
