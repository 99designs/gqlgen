package shardruntime

import (
	"context"
	"maps"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"

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
	ResolveExecutableComplexity(
		ctx context.Context,
		objectName string,
		fieldName string,
		childComplexity int,
		rawArgs map[string]any,
	) (int, bool)
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

type CodecMarshalHandler func(ctx context.Context, ec ObjectExecutionContext, sel ast.SelectionSet, value any) graphql.Marshaler

type CodecUnmarshalHandler func(ctx context.Context, ec ObjectExecutionContext, value any) (any, error)

var (
	mu                    sync.RWMutex
	objectByScope         = map[string]map[string]ObjectHandler{}
	streamByScope         = map[string]map[string]StreamObjectHandler{}
	fieldByScope          = map[string]map[string]map[string]FieldHandler{}
	streamFieldByScope    = map[string]map[string]map[string]StreamFieldHandler{}
	complexityByScope     = map[string]map[string]map[string]ComplexityHandler{}
	inputUnmarshalByScope = map[string]map[string]any{}
	codecMarshalByScope   = map[string]map[string]CodecMarshalHandler{}
	codecUnmarshalByScope = map[string]map[string]CodecUnmarshalHandler{}

	objectLookupSnapshot           atomic.Value
	streamObjectLookupSnapshot     atomic.Value
	fieldLookupSnapshot            atomic.Value
	fieldLookupSnapshotDirty       atomic.Bool
	streamFieldLookupSnapshot      atomic.Value
	complexityLookupSnapshot       atomic.Value
	inputUnmarshalMapByScopeLookup atomic.Value
	codecMarshalLookupSnapshot     atomic.Value
	codecUnmarshalLookupSnapshot   atomic.Value
)

var emptyInputUnmarshalMap = map[reflect.Type]reflect.Value{}

func init() {
	resetObjectLookupSnapshotForTest()
	resetStreamObjectLookupSnapshotForTest()
	resetFieldLookupSnapshotForTest()
	resetStreamFieldLookupSnapshotForTest()
	resetComplexityLookupSnapshotForTest()
	resetInputUnmarshalLookupSnapshotForTest()
	resetCodecMarshalLookupSnapshotForTest()
	resetCodecUnmarshalLookupSnapshotForTest()
}

func objectKey(scope, objectName string) string {
	return scope + "\x00" + objectName
}

func fieldKey(scope, objectName, fieldName string) string {
	return scope + "\x00" + objectName + "\x00" + fieldName
}

func codecKey(scope, funcName string) string {
	return scope + "\x00" + funcName
}

func cloneObjectHandlers(src map[string]ObjectHandler) map[string]ObjectHandler {
	return maps.Clone(src)
}

func cloneStreamObjectHandlers(src map[string]StreamObjectHandler) map[string]StreamObjectHandler {
	return maps.Clone(src)
}

func cloneStreamFieldHandlers(src map[string]StreamFieldHandler) map[string]StreamFieldHandler {
	return maps.Clone(src)
}

func cloneComplexityHandlers(src map[string]ComplexityHandler) map[string]ComplexityHandler {
	return maps.Clone(src)
}

func cloneInputUnmarshalMapByScope(src map[string]map[reflect.Type]reflect.Value) map[string]map[reflect.Type]reflect.Value {
	clone := make(map[string]map[reflect.Type]reflect.Value, len(src))
	for scope, handlers := range src {
		clone[scope] = handlers
	}
	return clone
}

func cloneInputUnmarshalHandlers(src map[reflect.Type]reflect.Value) map[reflect.Type]reflect.Value {
	return maps.Clone(src)
}

func cloneCodecMarshalHandlers(src map[string]CodecMarshalHandler) map[string]CodecMarshalHandler {
	return maps.Clone(src)
}

func cloneCodecUnmarshalHandlers(src map[string]CodecUnmarshalHandler) map[string]CodecUnmarshalHandler {
	return maps.Clone(src)
}

func loadObjectLookupSnapshot() map[string]ObjectHandler {
	if snapshot := objectLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]ObjectHandler)
	}
	return nil
}

func loadStreamObjectLookupSnapshot() map[string]StreamObjectHandler {
	if snapshot := streamObjectLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]StreamObjectHandler)
	}
	return nil
}

func loadFieldLookupSnapshot() map[string]FieldHandler {
	if snapshot := fieldLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]FieldHandler)
	}
	return nil
}

func loadStreamFieldLookupSnapshot() map[string]StreamFieldHandler {
	if snapshot := streamFieldLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]StreamFieldHandler)
	}
	return nil
}

func loadComplexityLookupSnapshot() map[string]ComplexityHandler {
	if snapshot := complexityLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]ComplexityHandler)
	}
	return nil
}

func loadInputUnmarshalLookupSnapshot() map[string]map[reflect.Type]reflect.Value {
	if snapshot := inputUnmarshalMapByScopeLookup.Load(); snapshot != nil {
		return snapshot.(map[string]map[reflect.Type]reflect.Value)
	}
	return nil
}

func loadCodecMarshalLookupSnapshot() map[string]CodecMarshalHandler {
	if snapshot := codecMarshalLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]CodecMarshalHandler)
	}
	return nil
}

func loadCodecUnmarshalLookupSnapshot() map[string]CodecUnmarshalHandler {
	if snapshot := codecUnmarshalLookupSnapshot.Load(); snapshot != nil {
		return snapshot.(map[string]CodecUnmarshalHandler)
	}
	return nil
}

func resetObjectLookupSnapshotForTest() {
	objectLookupSnapshot.Store(map[string]ObjectHandler{})
}

func resetStreamObjectLookupSnapshotForTest() {
	streamObjectLookupSnapshot.Store(map[string]StreamObjectHandler{})
}

func resetFieldLookupSnapshotForTest() {
	fieldLookupSnapshot.Store(map[string]FieldHandler{})
	fieldLookupSnapshotDirty.Store(false)
}

func resetStreamFieldLookupSnapshotForTest() {
	streamFieldLookupSnapshot.Store(map[string]StreamFieldHandler{})
}

func resetComplexityLookupSnapshotForTest() {
	complexityLookupSnapshot.Store(map[string]ComplexityHandler{})
}

func resetInputUnmarshalLookupSnapshotForTest() {
	inputUnmarshalMapByScopeLookup.Store(map[string]map[reflect.Type]reflect.Value{})
}

func resetCodecMarshalLookupSnapshotForTest() {
	codecMarshalLookupSnapshot.Store(map[string]CodecMarshalHandler{})
}

func resetCodecUnmarshalLookupSnapshotForTest() {
	codecUnmarshalLookupSnapshot.Store(map[string]CodecUnmarshalHandler{})
}

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

	lookup := cloneObjectHandlers(loadObjectLookupSnapshot())
	lookup[objectKey(scope, objectName)] = handler
	objectLookupSnapshot.Store(lookup)
}

func LookupObject(scope, objectName string) (ObjectHandler, bool) {
	handler, ok := loadObjectLookupSnapshot()[objectKey(scope, objectName)]
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

	lookup := cloneStreamObjectHandlers(loadStreamObjectLookupSnapshot())
	lookup[objectKey(scope, objectName)] = handler
	streamObjectLookupSnapshot.Store(lookup)
}

func LookupStreamObject(scope, objectName string) (StreamObjectHandler, bool) {
	handler, ok := loadStreamObjectLookupSnapshot()[objectKey(scope, objectName)]
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

	fieldLookupSnapshotDirty.Store(true)
}

func LookupField(scope, objectName, fieldName string) (FieldHandler, bool) {
	key := fieldKey(scope, objectName, fieldName)
	if fieldLookupSnapshotDirty.Load() {
		mu.Lock()
		if fieldLookupSnapshotDirty.Load() {
			rebuildFieldLookupSnapshotLocked()
		}
		mu.Unlock()
	}

	handler, ok := loadFieldLookupSnapshot()[key]
	return handler, ok
}

func rebuildFieldLookupSnapshotLocked() {
	totalFields := 0
	for _, scopeHandlers := range fieldByScope {
		for _, objectHandlers := range scopeHandlers {
			totalFields += len(objectHandlers)
		}
	}

	lookup := make(map[string]FieldHandler, totalFields)
	for scope, scopeHandlers := range fieldByScope {
		for objectName, objectHandlers := range scopeHandlers {
			for fieldName, handler := range objectHandlers {
				lookup[fieldKey(scope, objectName, fieldName)] = handler
			}
		}
	}

	fieldLookupSnapshot.Store(lookup)
	fieldLookupSnapshotDirty.Store(false)
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

	lookup := cloneStreamFieldHandlers(loadStreamFieldLookupSnapshot())
	lookup[fieldKey(scope, objectName, fieldName)] = handler
	streamFieldLookupSnapshot.Store(lookup)
}

func LookupStreamField(scope, objectName, fieldName string) (StreamFieldHandler, bool) {
	handler, ok := loadStreamFieldLookupSnapshot()[fieldKey(scope, objectName, fieldName)]
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

	lookup := cloneComplexityHandlers(loadComplexityLookupSnapshot())
	lookup[fieldKey(scope, objectName, fieldName)] = handler
	complexityLookupSnapshot.Store(lookup)
}

func LookupComplexity(scope, objectName, fieldName string) (ComplexityHandler, bool) {
	handler, ok := loadComplexityLookupSnapshot()[fieldKey(scope, objectName, fieldName)]
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

	ft := reflect.TypeOf(fn)
	if ft == nil || ft.Kind() != reflect.Func || ft.NumOut() == 0 {
		return
	}

	lookup := cloneInputUnmarshalMapByScope(loadInputUnmarshalLookupSnapshot())
	inputLookupByType := cloneInputUnmarshalHandlers(lookup[scope])
	if inputLookupByType == nil {
		inputLookupByType = map[reflect.Type]reflect.Value{}
	}
	inputLookupByType[ft.Out(0)] = reflect.ValueOf(fn)
	lookup[scope] = inputLookupByType
	inputUnmarshalMapByScopeLookup.Store(lookup)
}

func InputUnmarshalMap(scope string, _ ObjectExecutionContext) map[reflect.Type]reflect.Value {
	scopeHandlers := loadInputUnmarshalLookupSnapshot()[scope]
	if scopeHandlers == nil {
		return emptyInputUnmarshalMap
	}

	return scopeHandlers
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

func RegisterCodecMarshal(scope, funcName string, handler CodecMarshalHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := codecMarshalByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]CodecMarshalHandler{}
		codecMarshalByScope[scope] = scopeHandlers
	}

	if _, exists := scopeHandlers[funcName]; exists {
		panic("duplicate codec marshal handler registration: " + scope + ":" + funcName)
	}
	scopeHandlers[funcName] = handler

	lookup := cloneCodecMarshalHandlers(loadCodecMarshalLookupSnapshot())
	lookup[codecKey(scope, funcName)] = handler
	codecMarshalLookupSnapshot.Store(lookup)
}

func LookupCodecMarshal(scope, funcName string) (CodecMarshalHandler, bool) {
	handler, ok := loadCodecMarshalLookupSnapshot()[codecKey(scope, funcName)]
	return handler, ok
}

func RegisterCodecUnmarshal(scope, funcName string, handler CodecUnmarshalHandler) {
	mu.Lock()
	defer mu.Unlock()

	scopeHandlers := codecUnmarshalByScope[scope]
	if scopeHandlers == nil {
		scopeHandlers = map[string]CodecUnmarshalHandler{}
		codecUnmarshalByScope[scope] = scopeHandlers
	}

	if _, exists := scopeHandlers[funcName]; exists {
		panic("duplicate codec unmarshal handler registration: " + scope + ":" + funcName)
	}
	scopeHandlers[funcName] = handler

	lookup := cloneCodecUnmarshalHandlers(loadCodecUnmarshalLookupSnapshot())
	lookup[codecKey(scope, funcName)] = handler
	codecUnmarshalLookupSnapshot.Store(lookup)
}

func LookupCodecUnmarshal(scope, funcName string) (CodecUnmarshalHandler, bool) {
	handler, ok := loadCodecUnmarshalLookupSnapshot()[codecKey(scope, funcName)]
	return handler, ok
}
