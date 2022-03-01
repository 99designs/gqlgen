// plugin package interfaces are EXPERIMENTAL.

package plugin

import (
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/v2/ast"
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

// EarlySourceInjector is used to inject things that are required for user schema files to compile.
// Deprecated: Use EarlySourcesInjector instead
type EarlySourceInjector interface {
	InjectSourceEarly() *ast.Source
}

// EarlySourcesInjector is used to inject things that are required for user schema files to compile.
type EarlySourcesInjector interface {
	InjectSourcesEarly() ([]*ast.Source, error)
}

// LateSourceInjector is used to inject more sources, after we have loaded the users schema.
// Deprecated: Use LateSourcesInjector instead
type LateSourceInjector interface {
	InjectSourceLate(schema *ast.Schema) *ast.Source
}

// LateSourcesInjector is used to inject more sources, after we have loaded the users schema.
type LateSourcesInjector interface {
	InjectSourcesLate(schema *ast.Schema) ([]*ast.Source, error)
}
