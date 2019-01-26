// plugin package interfaces are EXPERIMENTAL.

package plugin

import "github.com/99designs/gqlgen/codegen/config"

type Plugin interface {
	Name() string
}

type ConfigMutator interface {
	MutateConfig(cfg *config.Config) error
}
