package codegen

import (
	"github.com/vektah/gqlparser/ast"
)

// Plugin is an interface for a gqlgen plugin
type Plugin interface {
	Name() string
}

// PluginConfigurer is an interface a plugin can satisfy in order to make changes to configuration before codegen
type PluginConfigurer interface {
	PostNormalize(c *Config, schema *ast.Schema) error
}

// PluginSchema is an interface a plugin can satisfy if they wish to merge additional schema with the base schema
type PluginSchema interface {
	Schema(cfg *Config) (string, error)
}

// PluginRegistry is a collection of plugins that will be accessed during codegen
type PluginRegistry struct {
	plugins []Plugin
}

// Register a set of plugins to the plugin registry
func (r *PluginRegistry) Register(p ...Plugin) {
	r.plugins = append(r.plugins, p...)
}

func (r *PluginRegistry) schemas(c *Config) (srcs []*ast.Source, err error) {
	for _, p := range r.plugins {
		name := p.Name()
		if p, ok := p.(PluginSchema); ok {
			src, err := p.Schema(c)
			if err != nil {
				return nil, err
			}
			srcs = append(srcs, &ast.Source{Name: name, Input: src})
		}
	}
	return srcs, err
}

func (r *PluginRegistry) postNormalize(cfg *Config, schema *ast.Schema) error {
	for _, p := range r.plugins {
		if p, ok := p.(PluginConfigurer); ok {
			err := p.PostNormalize(cfg, schema)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
