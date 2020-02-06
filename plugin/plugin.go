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

// This should be used to inject directives etc. Things that are required for user schema files to compile
type EarlySourcesInjector interface {
	InjectSourcesEarly() *ast.Source
}

// This hook runs a bit later, after the schema is valid. This allows you to reflect on the users schema
// and inject more, like a relay Node union.
type LateSourcesInjector interface {
	InjectSourcesLate(schema *ast.Schema) *ast.Source
}
