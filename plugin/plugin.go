// plugin package interfaces are EXPERIMENTAL.

package plugin

import (
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
)

type Plugin interface {
	Name() string
}

// SchemaMutator is used to modify the schema before it is used to generate code
// Similarly to [ConfigMutator] that is also triggered before code generation, SchemaMutator
// can be used to modify the schema even before the models are generated.
type SchemaMutator interface {
	MutateSchema(schema *ast.Schema) error
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

// ResolverImplementer is used to generate code inside resolvers
type ResolverImplementer interface {
	Implement(prevImplementation string, field *codegen.Field) string
}

// LateSourcesInjector is used to inject more sources, after we have loaded the users schema.
type LateSourcesInjector interface {
	InjectSourcesLate(schema *ast.Schema) ([]*ast.Source, error)
}
