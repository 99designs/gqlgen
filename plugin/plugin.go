// plugin package interfaces are EXPERIMENTAL.

package plugin

import (
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/ast"
)

type Plugin interface {
	Name() string
}

type ConfigMutator interface {
	MutateConfig(cfg *config.Config) error
}

type CodeGenerator interface {
	GenerateCode(cfg *codegen.Data) error
}

type SourcesInjector interface {
	InjectSources(cfg *config.Config)
}

type SchemaMutator interface {
	MutateSchema(s *ast.Schema) error
}
