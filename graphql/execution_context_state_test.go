package graphql

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

type (
	testResolverRoot   struct{}
	testDirectiveRoot  struct{}
	testComplexityRoot struct{}
)

func newTestExecutionContextState(
	opCtx *OperationContext,
	schemaData *ast.Schema,
	parsedSchema *ast.Schema,
	deferredResults chan DeferredResult,
) *ExecutionContextState[testResolverRoot, testDirectiveRoot, testComplexityRoot] {
	if opCtx == nil {
		opCtx = &OperationContext{}
	}
	if deferredResults == nil {
		deferredResults = make(chan DeferredResult, 1)
	}
	return NewExecutionContextState[testResolverRoot, testDirectiveRoot, testComplexityRoot](
		opCtx,
		&ExecutableSchemaState[testResolverRoot, testDirectiveRoot, testComplexityRoot]{
			SchemaData: schemaData,
		},
		parsedSchema,
		deferredResults,
	)
}

func receiveDeferredResult(t *testing.T, ch <-chan DeferredResult) DeferredResult {
	t.Helper()

	select {
	case res := <-ch:
		return res
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for deferred result")
		return DeferredResult{}
	}
}

func makeSchemaWithType(typeName string) *ast.Schema {
	query := &ast.Definition{Name: "Query", Kind: ast.Object}
	typ := &ast.Definition{Name: typeName, Kind: ast.Object}

	return &ast.Schema{
		Query: query,
		Types: map[string]*ast.Definition{
			"Query":  query,
			typeName: typ,
		},
		Directives: map[string]*ast.DirectiveDefinition{},
	}
}

func TestExecutionContextState_Schema(t *testing.T) {
	schemaData := makeSchemaWithType("SchemaDataType")
	parsedSchema := makeSchemaWithType("ParsedType")

	ec := newTestExecutionContextState(
		&OperationContext{},
		schemaData,
		parsedSchema,
		nil,
	)

	assert.Same(t, schemaData, ec.Schema())
}

func TestExecutionContextState_Schema_FallsBackToParsedSchema(t *testing.T) {
	parsedSchema := makeSchemaWithType("ParsedType")

	ec := newTestExecutionContextState(
		&OperationContext{},
		nil,
		parsedSchema,
		nil,
	)

	assert.Same(t, parsedSchema, ec.Schema())
}

func TestExecutionContextState_IntrospectionDisabled(t *testing.T) {
	ec := newTestExecutionContextState(
		&OperationContext{DisableIntrospection: true},
		nil,
		makeSchemaWithType("Foo"),
		nil,
	)

	schema, schemaErr := ec.IntrospectSchema()
	require.Error(t, schemaErr)
	require.EqualError(t, schemaErr, "introspection disabled")
	assert.Nil(t, schema)

	typ, typeErr := ec.IntrospectType("Foo")
	require.Error(t, typeErr)
	require.EqualError(t, typeErr, "introspection disabled")
	assert.Nil(t, typ)
}

func TestExecutionContextState_IntrospectType(t *testing.T) {
	ec := newTestExecutionContextState(
		&OperationContext{},
		nil,
		makeSchemaWithType("Foo"),
		nil,
	)

	typ, err := ec.IntrospectType("Foo")
	require.NoError(t, err)
	require.NotNil(t, typ)

	name := typ.Name()
	require.NotNil(t, name)
	assert.Equal(t, "Foo", *name)

	missing, missingErr := ec.IntrospectType("Missing")
	require.NoError(t, missingErr)
	assert.Nil(t, missing)
}

func TestExecutionContextState_ProcessDeferredGroup_IncrementsPendingAndPropagates(t *testing.T) {
	deferredResults := make(chan DeferredResult, 1)
	ec := newTestExecutionContextState(
		&OperationContext{},
		nil,
		makeSchemaWithType("Foo"),
		deferredResults,
	)

	ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, DefaultRecover)
	fieldSet := NewFieldSet([]CollectedField{{Field: &ast.Field{Alias: "value"}}})
	fieldSet.Concurrently(0, func(ctx context.Context) Marshaler {
		return MarshalString("ok")
	})
	view := fieldSet.NewView()
	view.AddIndices(0)

	path := ast.Path{ast.PathName("query")}
	label := "group-1"

	ec.ProcessDeferredGroup(DeferredGroup{
		Defers: map[string]*FieldSetView{
			label: view,
		},
		Path:     path,
		FieldSet: fieldSet,
		Context:  ctx,
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&ec.PendingDeferred))

	result := receiveDeferredResult(t, deferredResults)
	assert.Equal(t, path, result.Path)
	assert.Equal(t, label, result.Label)
	assert.Same(t, view, result.Result)
	assert.Nil(t, result.Errors)
}

// The task is scheduled with Concurrently and keeps its own Invalids count,
// as code generated before ResolveConcurrently does. Its payload must still
// null, or that code would silently stop bubbling deferred non-null errors.
// Remove with the compatibility fallback in FieldSetView.payloadInvalid.
func TestExecutionContextState_ProcessDeferredGroup_NullsOnInvalidAndIsolatesErrors(t *testing.T) {
	deferredResults := make(chan DeferredResult, 1)
	ec := newTestExecutionContextState(
		&OperationContext{},
		nil,
		makeSchemaWithType("Foo"),
		deferredResults,
	)

	ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, DefaultRecover)
	AddError(ctx, errors.New("parent error"))

	fieldSet := NewFieldSet([]CollectedField{{Field: &ast.Field{Alias: "value"}}})
	fieldSet.Concurrently(0, func(ctx context.Context) Marshaler {
		AddError(ctx, errors.New("deferred error"))
		atomic.AddUint32(&fieldSet.Invalids, 1)
		return MarshalString("ignored")
	})

	view := fieldSet.NewView()
	view.AddIndices(0)
	ec.ProcessDeferredGroup(DeferredGroup{
		Path:     ast.Path{ast.PathName("query")},
		Defers:   map[string]*FieldSetView{"group-2": view},
		FieldSet: fieldSet,
		Context:  ctx,
	})

	result := receiveDeferredResult(t, deferredResults)
	assert.Same(t, Null, result.Result)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "deferred error", result.Errors[0].Message)

	parentErrors := GetErrors(ctx)
	require.Len(t, parentErrors, 1)
	assert.Equal(t, "parent error", parentErrors[0].Message)
}

// Each @defer label is its own incremental payload, so an invalid non-null
// field under one label nulls that label's payload and no other. The valid
// label is delivered after the invalid one, the order in which a count shared
// by the whole group would null both.
func TestExecutionContextState_ProcessDeferredGroup_NullsOnlyTheInvalidLabel(t *testing.T) {
	t.Parallel()

	ctx, oc := newResolveConcurrentlyTestContext()
	deferredResults := make(chan DeferredResult, 2)
	ec := newTestExecutionContextState(oc, nil, makeSchemaWithType("Foo"), deferredResults)

	out := NewFieldSet([]CollectedField{
		collectedField("valid", "labelA"),
		collectedField("invalid", "labelB"),
	})
	deferred := NewDeferredGroup(ctx)
	release := make(chan struct{})
	out.ResolveConcurrently(oc, &deferred, 0, true, true, func(context.Context) Marshaler {
		<-release
		return MarshalString("ok")
	})
	out.ResolveConcurrently(oc, &deferred, 1, true, true,
		func(context.Context) Marshaler { return Null })

	ec.ProcessDeferredGroup(deferred)

	invalid := receiveDeferredResult(t, deferredResults)
	close(release)
	valid := receiveDeferredResult(t, deferredResults)

	assert.Equal(t, "labelB", invalid.Label)
	assert.Same(t, Null, invalid.Result, "the invalid label's payload is null")

	assert.Equal(t, "labelA", valid.Label)
	var b bytes.Buffer
	valid.Result.MarshalGQL(&b)
	assert.JSONEq(t, `{"valid":"ok"}`, b.String(), "the valid label's payload is delivered")
}
