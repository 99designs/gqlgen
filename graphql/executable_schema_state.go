package graphql

import (
	"reflect"

	"github.com/vektah/gqlparser/v2/ast"
)

// ExecutableSchemaState stores generated executable schema dependencies.
// Generated code defines its local executableSchema type from this one.
type ExecutableSchemaState[R any, D any, C any] struct {
	SchemaData     *ast.Schema
	Resolvers      R
	Directives     D
	ComplexityRoot C

	// InputUnmarshalers is the schema's static input-type unmarshaler map,
	// built once at construction and reused across every operation.
	InputUnmarshalers map[reflect.Type]reflect.Value
}
