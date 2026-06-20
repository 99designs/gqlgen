package graphql

import (
	"context"
	"errors"
	"sync"
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

// ProcessDeferredGroup resolves the deferred fields owned by dg and sends one
// DeferredResult per view to ec.DeferredResults.
//
// Errors accumulate append-only on the group's shared response context. Each
// view's DeferredResult.Errors holds the suffix of errors that arrived after
// the previous view in this group completed, so every error appears in
// exactly one view's payload. This matters when two @defer fragments share a
// field that errors: without partitioning, the error would be duplicated
// across labels.
func (ec *ExecutionContextState[R, D, C]) ProcessDeferredGroup(dg DeferredGroup) {
	if len(dg.Defers) == 0 {
		return
	}

	atomic.AddInt32(&ec.PendingDeferred, int32(len(dg.Defers)))
	ctx := WithFreshResponseContext(dg.Context)

	var (
		errorsMu      sync.Mutex
		errorsEmitted int
	)

	for label, view := range dg.Defers {
		view.SetOnComplete(func(ctx context.Context) {
			errorsMu.Lock()
			allErrors := GetErrors(ctx)
			errs := allErrors[errorsEmitted:]
			errorsEmitted = len(allErrors)
			errorsMu.Unlock()

			ds := DeferredResult{
				Path:   dg.Path,
				Label:  label,
				Result: view,
				Errors: errs,
			}
			// null fields should bubble up
			if dg.FieldSet.Invalids > 0 {
				ds.Result = Null
			}
			select {
			case ec.DeferredResults <- ds:
			case <-ctx.Done():
			}
		})
	}

	go func() {
		dg.FieldSet.Dispatch(ctx)
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
