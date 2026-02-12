package shardruntime

import (
	"context"
	"sort"
	"sync"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// ObjectExecutionContext defines the runtime surface required by generated object shards.
type ObjectExecutionContext interface {
	GetOperationContext() *graphql.OperationContext
	ResolveExecutableField(
		ctx context.Context,
		objectName string,
		fieldName string,
		field graphql.CollectedField,
		obj any,
	) graphql.Marshaler
	ResolveExecutableStreamField(
		ctx context.Context,
		objectName string,
		fieldName string,
		field graphql.CollectedField,
		obj any,
	) func(context.Context) graphql.Marshaler
	MarshalCodec(
		ctx context.Context,
		funcName string,
		sel ast.SelectionSet,
		value any,
	) graphql.Marshaler
	UnmarshalCodec(
		ctx context.Context,
		funcName string,
		value any,
	) (any, error)
	ParseFieldArgs(
		ctx context.Context,
		argsKey string,
		rawArgs map[string]any,
	) (map[string]any, error)
	ResolveField(
		ctx context.Context,
		objectName string,
		fieldName string,
		field graphql.CollectedField,
		obj any,
	) graphql.Marshaler
	ResolveStreamField(
		ctx context.Context,
		objectName string,
		fieldName string,
		field graphql.CollectedField,
		obj any,
	) func(context.Context) graphql.Marshaler
	ProcessDeferredGroup(dg graphql.DeferredGroup)
	AddDeferred(delta int32)
	Error(ctx context.Context, err error)
	Recover(ctx context.Context, err any) error
}

type ObjectHandler func(
	ctx context.Context,
	ec ObjectExecutionContext,
	sel ast.SelectionSet,
	obj any,
) graphql.Marshaler

type StreamObjectHandler func(
	ctx context.Context,
	ec ObjectExecutionContext,
	sel ast.SelectionSet,
) func(context.Context) graphql.Marshaler

type FieldHandler func(
	ctx context.Context,
	ec ObjectExecutionContext,
	field graphql.CollectedField,
	obj any,
) graphql.Marshaler

type StreamFieldHandler func(
	ctx context.Context,
	ec ObjectExecutionContext,
	field graphql.CollectedField,
	obj any,
) func(context.Context) graphql.Marshaler

type ComplexityHandler func(
	ctx context.Context,
	ec ObjectExecutionContext,
	childComplexity int,
	rawArgs map[string]any,
) (int, bool)

var (
	mu                    sync.RWMutex
	objectByScope         = map[string]map[string]ObjectHandler{}
	streamByScope         = map[string]map[string]StreamObjectHandler{}
	fieldByScope          = map[string]map[string]map[string]FieldHandler{}
	streamFieldByScope    = map[string]map[string]map[string]StreamFieldHandler{}
	complexityByScope     = map[string]map[string]map[string]ComplexityHandler{}
	inputUnmarshalByScope = map[string]map[string]any{}
)

func RegisterObject(scope, objectName string, handler ObjectHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := objectByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]ObjectHandler{}
		objectByScope[scope] = scopeHandlers
	}

	if _, exists := scopeHandlers[objectName]; exists {
		panic("duplicate object shard handler registration: " + scope + ":" + objectName)
	}
	scopeHandlers[objectName] = handler
}

func LookupObject(scope, objectName string) (ObjectHandler, bool) {
	mu.RLock()
	defer mu.RUnlock()

	scopeHandlers := objectByScope[scope]
	if scopeHandlers == nil {
		return nil, false
	}
	handler, ok := scopeHandlers[objectName]
	return handler, ok
}

func RegisterStreamObject(scope, objectName string, handler StreamObjectHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := streamByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]StreamObjectHandler{}
		streamByScope[scope] = scopeHandlers
	}

	if _, exists := scopeHandlers[objectName]; exists {
		panic("duplicate stream object shard handler registration: " + scope + ":" + objectName)
	}
	scopeHandlers[objectName] = handler
}

func LookupStreamObject(scope, objectName string) (StreamObjectHandler, bool) {
	mu.RLock()
	defer mu.RUnlock()

	scopeHandlers := streamByScope[scope]
	if scopeHandlers == nil {
		return nil, false
	}
	handler, ok := scopeHandlers[objectName]
	return handler, ok
}

func RegisterField(scope, objectName, fieldName string, handler FieldHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := fieldByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]map[string]FieldHandler{}
		fieldByScope[scope] = scopeHandlers
	}

	objectHandlers := scopeHandlers[objectName]
	if objectHandlers == nil {
		objectHandlers = map[string]FieldHandler{}
		scopeHandlers[objectName] = objectHandlers
	}

	if _, exists := objectHandlers[fieldName]; exists {
		panic("duplicate field shard handler registration: " + scope + ":" + objectName + ":" + fieldName)
	}
	objectHandlers[fieldName] = handler
}

func LookupField(scope, objectName, fieldName string) (FieldHandler, bool) {
	mu.RLock()
	defer mu.RUnlock()

	scopeHandlers := fieldByScope[scope]
	if scopeHandlers == nil {
		return nil, false
	}

	objectHandlers := scopeHandlers[objectName]
	if objectHandlers == nil {
		return nil, false
	}

	handler, ok := objectHandlers[fieldName]
	return handler, ok
}

func RegisterStreamField(scope, objectName, fieldName string, handler StreamFieldHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := streamFieldByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]map[string]StreamFieldHandler{}
		streamFieldByScope[scope] = scopeHandlers
	}

	objectHandlers := scopeHandlers[objectName]
	if objectHandlers == nil {
		objectHandlers = map[string]StreamFieldHandler{}
		scopeHandlers[objectName] = objectHandlers
	}

	if _, exists := objectHandlers[fieldName]; exists {
		panic("duplicate stream field shard handler registration: " + scope + ":" + objectName + ":" + fieldName)
	}
	objectHandlers[fieldName] = handler
}

func LookupStreamField(scope, objectName, fieldName string) (StreamFieldHandler, bool) {
	mu.RLock()
	defer mu.RUnlock()

	scopeHandlers := streamFieldByScope[scope]
	if scopeHandlers == nil {
		return nil, false
	}

	objectHandlers := scopeHandlers[objectName]
	if objectHandlers == nil {
		return nil, false
	}

	handler, ok := objectHandlers[fieldName]
	return handler, ok
}

func RegisterComplexity(scope, objectName, fieldName string, handler ComplexityHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := complexityByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]map[string]ComplexityHandler{}
		complexityByScope[scope] = scopeHandlers
	}

	objectHandlers := scopeHandlers[objectName]
	if objectHandlers == nil {
		objectHandlers = map[string]ComplexityHandler{}
		scopeHandlers[objectName] = objectHandlers
	}

	if _, exists := objectHandlers[fieldName]; exists {
		panic("duplicate complexity shard handler registration: " + scope + ":" + objectName + ":" + fieldName)
	}
	objectHandlers[fieldName] = handler
}

func LookupComplexity(scope, objectName, fieldName string) (ComplexityHandler, bool) {
	mu.RLock()
	defer mu.RUnlock()

	scopeHandlers := complexityByScope[scope]
	if scopeHandlers == nil {
		return nil, false
	}

	objectHandlers := scopeHandlers[objectName]
	if objectHandlers == nil {
		return nil, false
	}

	handler, ok := objectHandlers[fieldName]
	return handler, ok
}

func RegisterInputUnmarshaler(scope, inputName string, fn any) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := inputUnmarshalByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]any{}
		inputUnmarshalByScope[scope] = scopeHandlers
	}

	if _, exists := scopeHandlers[inputName]; exists {
		panic("duplicate input unmarshaler registration: " + scope + ":" + inputName)
	}
	scopeHandlers[inputName] = fn
}

func ListInputUnmarshalers(scope string, _ ObjectExecutionContext) []any {
	mu.RLock()
	defer mu.RUnlock()

	scopeHandlers := inputUnmarshalByScope[scope]
	if scopeHandlers == nil {
		return nil
	}

	inputNames := make([]string, 0, len(scopeHandlers))
	for inputName := range scopeHandlers {
		inputNames = append(inputNames, inputName)
	}
	sort.Strings(inputNames)

	inputUnmarshalers := make([]any, 0, len(scopeHandlers))
	for _, inputName := range inputNames {
		inputUnmarshalers = append(inputUnmarshalers, scopeHandlers[inputName])
	}

	return inputUnmarshalers
}
