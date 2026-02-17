package graphql

import (
	"errors"
	"sync/atomic"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql/introspection"
)

// ExecutionContextState stores generated execution context dependencies and state.
// Generated code defines its local executionContext type from this one.
type ExecutionContextState[R any, D any, C any] struct {
	*OperationContext
	*ExecutableSchemaState[R, D, C]
	ParsedSchema    *ast.Schema
	Deferred        int32
	PendingDeferred int32
	DeferredResults chan DeferredResult
}

func NewExecutionContextState[R any, D any, C any](
	operationContext *OperationContext,
	executableSchemaState *ExecutableSchemaState[R, D, C],
	parsedSchema *ast.Schema,
	deferredResults chan DeferredResult,
) *ExecutionContextState[R, D, C] {
	return &ExecutionContextState[R, D, C]{
		OperationContext:      operationContext,
		ExecutableSchemaState: executableSchemaState,
		ParsedSchema:          parsedSchema,
		DeferredResults:       deferredResults,
	}
}

func (ec *ExecutionContextState[R, D, C]) Schema() *ast.Schema {
	if ec.SchemaData != nil {
		return ec.SchemaData
	}
	return ec.ParsedSchema
}

func (ec *ExecutionContextState[R, D, C]) ProcessDeferredGroup(dg DeferredGroup) {
	atomic.AddInt32(&ec.PendingDeferred, 1)
	go func() {
		ctx := WithFreshResponseContext(dg.Context)
		dg.FieldSet.Dispatch(ctx)
		ds := DeferredResult{
			Path:   dg.Path,
			Label:  dg.Label,
			Result: dg.FieldSet,
			Errors: GetErrors(ctx),
		}
		// null fields should bubble up
		if dg.FieldSet.Invalids > 0 {
			ds.Result = Null
		}
		ec.DeferredResults <- ds
	}()
}

func (ec *ExecutionContextState[R, D, C]) IntrospectSchema() (*introspection.Schema, error) {
	if ec.DisableIntrospection {
		return nil, errors.New("introspection disabled")
	}
	return introspection.WrapSchema(ec.Schema()), nil
}

func (ec *ExecutionContextState[R, D, C]) IntrospectType(name string) (*introspection.Type, error) {
	if ec.DisableIntrospection {
		return nil, errors.New("introspection disabled")
	}
	return introspection.WrapTypeFromDef(ec.Schema(), ec.Schema().Types[name]), nil
}
