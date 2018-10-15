package plugins

import (
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/plugins/resolver"
)

// DefaultPlugins is a slice of the default set of plugins gqlgen operates with
func DefaultPlugins() []codegen.Plugin {
	return []codegen.Plugin{
		&resolver.Plugin{},
	}
}
