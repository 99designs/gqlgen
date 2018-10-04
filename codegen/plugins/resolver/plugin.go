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

func (p *Plugin) Schema(cfg *codegen.Config) (*ast.Source, error) {
	return &ast.Source{
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
			if d := f.Directives.ForName("resolver"); d == nil {
				continue
			}
			modelCfg := cfg.Models[typ.Name]
			if modelCfg.Fields == nil {
				modelCfg.Fields = make(map[string]codegen.TypeMapField)
			}
			fieldCfg := modelCfg.Fields[f.Name]
			fieldCfg.ForceResolver = true
			modelCfg.Fields[f.Name] = fieldCfg
			cfg.Models[typ.Name] = modelCfg
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
