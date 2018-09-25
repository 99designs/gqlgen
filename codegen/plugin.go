package codegen

import (
	"github.com/vektah/gqlparser/ast"
)

type Plugin interface {
	Execute(c *Config, schema *ast.Schema) error
	Schema(cfg *Config) (*ast.Source, error)
}

type PluginRegistry struct {
	plugins []Plugin
}

var DefaultPluginRegistry = PluginRegistry{}

// RegisterPlugin registers a plugin in the default plugin register
func RegisterPlugin(p Plugin) {

	DefaultPluginRegistry.Register(p)
}

func (r *PluginRegistry) Register(p Plugin) {
	r.plugins = append(r.plugins, p)
}

func (r *PluginRegistry) Schemas(c *Config) (srcs []*ast.Source, err error) {
	for _, p := range r.plugins {
		src, err := p.Schema(c)
		if err != nil {
			return nil, err
		}
		if src != nil {
			srcs = append(srcs, src)
		}
	}
	return srcs, err
}

func (r *PluginRegistry) Execute(cfg *Config, schema *ast.Schema) error {
	for _, p := range r.plugins {
		if err := p.Execute(cfg, schema); err != nil {
			return err
		}
	}
	return nil
}
