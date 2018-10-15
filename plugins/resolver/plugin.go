package resolver

import (
	"context"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/graphql"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

// Plugin is a codegen plugin for adding force resolver directive
type Plugin struct{}

// Name returns the name of the plugin
func (p *Plugin) Name() string {
	return "resolver"
}

// Schema returns a schema containing the @resolver directive
func (p *Plugin) Schema(cfg *codegen.Config) (string, error) {
	return "directive @resolver on FIELD_DEFINITION", nil
}

// Execute updates the provided config to force resolvers when the @resolver directive is encountered
func (p *Plugin) Execute(cfg *codegen.Config, schema *ast.Schema) error {
	if cfg.Directives.ImplementationFor("resolver") != "" {
		return errors.Errorf("directive implementation for resolver already exists")
	}
	cfg.Directives["resolver"] = codegen.DirectiveMapEntry{Implementation: "github.com/99designs/gqlgen/plugins/resolver.DirectiveNoop"}
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

// DirectiveNoop is a no-op directive implementation to prevent codegen generating a directive middleware for @resolver
func DirectiveNoop(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	return next(ctx)
}
