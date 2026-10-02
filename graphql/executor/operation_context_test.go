package executor_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
)

// CreateOperationContext turns the raw parameters a transport decoded into the
// OperationContext the rest of a request runs against. Everything downstream
// reads that struct rather than the parameters, so a field it fails to copy is
// a field that is simply absent for the whole request: a lost OperationName
// selects the wrong operation, lost Headers hide authentication from
// extensions, and a lost RawQuery breaks both the query cache key and tracing.
//
// Nothing in the executor reads those fields back, which is why they need
// asserting here rather than being covered incidentally by an execution test.

// paramMutator is an extension that rewrites or rejects the raw parameters
// before the operation context is built. Func fields and an Invoked flag follow
// the hand-written mock pattern, so a test can assert both what happened and
// whether the hook ran at all.
type paramMutator struct {
	MutateFunc func(ctx context.Context, params *graphql.RawParams) *gqlerror.Error
	Invoked    bool
	SawOpCtx   bool
}

func (*paramMutator) ExtensionName() string { return "testParamMutator" }

func (*paramMutator) Validate(graphql.ExecutableSchema) error { return nil }

func (m *paramMutator) MutateOperationParameters(
	ctx context.Context,
	params *graphql.RawParams,
) *gqlerror.Error {
	m.Invoked = true
	// The operation context is placed on the context before this hook runs, so
	// a mutator can reach it. Recording that is how the test pins the ordering.
	m.SawOpCtx = graphql.HasOperationContext(ctx)
	if m.MutateFunc == nil {
		return nil
	}
	return m.MutateFunc(ctx, params)
}

// ctxMutator is an extension that adjusts or rejects the operation context
// after it has been built and validated.
type ctxMutator struct {
	MutateFunc func(ctx context.Context, opCtx *graphql.OperationContext) *gqlerror.Error
	Invoked    bool
}

func (*ctxMutator) ExtensionName() string { return "testCtxMutator" }

func (*ctxMutator) Validate(graphql.ExecutableSchema) error { return nil }

func (m *ctxMutator) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	m.Invoked = true
	if m.MutateFunc == nil {
		return nil
	}
	return m.MutateFunc(ctx, opCtx)
}

// createOpCtx runs CreateOperationContext against params, failing the test if
// it reports errors.
func createOpCtx(
	t *testing.T,
	exec *testexecutor.TestExecutor,
	params *graphql.RawParams,
) *graphql.OperationContext {
	t.Helper()

	opCtx, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()), params)
	require.Empty(t, errs)
	require.NotNil(t, opCtx)
	return opCtx
}

// Every raw parameter the transport decoded has to reach the operation context.
func TestCreateOperationContextCopiesTheRawParameters(t *testing.T) {
	t.Parallel()

	headers := http.Header{"Authorization": []string{"Bearer token"}}
	extensions := map[string]any{"persistedQuery": map[string]any{"version": 1}}
	const (
		q      = "query Named($id: Int!) { find(id: $id) }"
		opName = "Named"
	)

	opCtx := createOpCtx(t, testexecutor.New(), &graphql.RawParams{
		Query:         q,
		OperationName: opName,
		Variables:     map[string]any{"id": 1},
		Extensions:    extensions,
		Headers:       headers,
	})

	assert.Equal(t, q, opCtx.RawQuery, "RawQuery feeds the query cache key and tracing")
	assert.Equal(t, opName, opCtx.OperationName, "OperationName selects the operation")
	assert.Equal(t, extensions, opCtx.Extensions, "Extensions carry APQ and client data")
	assert.Equal(t, headers, opCtx.Headers, "Headers carry authentication to extensions")
}

// The operation context starts with introspection disabled and the executor's
// own recover and middleware functions installed. An extension that enables
// introspection relies on the default being off, so the default is a contract.
func TestCreateOperationContextDefaults(t *testing.T) {
	t.Parallel()

	opCtx := createOpCtx(t, testexecutor.New(), &graphql.RawParams{Query: "{ name }"})

	assert.True(t, opCtx.DisableIntrospection,
		"introspection is off unless an extension turns it on")
	assert.NotNil(t, opCtx.RecoverFunc, "a panicking resolver needs a recover function")
	assert.NotNil(t, opCtx.ResolverMiddleware, "field middleware must be installed")
	assert.NotNil(t, opCtx.RootResolverMiddleware, "root field middleware must be installed")
}

// Variables are coerced against the schema and the operation's declarations,
// so the operation context carries typed values rather than the raw JSON the
// transport decoded.
func TestCreateOperationContextCoercesVariables(t *testing.T) {
	t.Parallel()

	opCtx := createOpCtx(t, testexecutor.New(), &graphql.RawParams{
		Query:     "query Find($id: Int!) { find(id: $id) }",
		Variables: map[string]any{"id": 42},
	})

	require.NotNil(t, opCtx.Variables)
	// Coercion produces a Go int for a GraphQL Int, which is the concrete type
	// a generated resolver then unmarshals from.
	assert.Equal(t, 42, opCtx.Variables["id"],
		"an Int variable should be coerced to a Go int")
}

// A variable that does not satisfy its declaration is a client error, and it
// has to be reported as a validation failure rather than as an unlabelled
// internal one, because that is what a client and the error metrics key on.
func TestCreateOperationContextRejectsBadVariables(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query     string
		variables map[string]any
	}{
		"a required variable is missing": {
			query:     "query Find($id: Int!) { find(id: $id) }",
			variables: map[string]any{},
		},
		"a variable has the wrong type": {
			query:     "query Find($id: Int!) { find(id: $id) }",
			variables: map[string]any{"id": "not an int"},
		},
		"a required variable is null": {
			query:     "query Find($id: Int!) { find(id: $id) }",
			variables: map[string]any{"id": nil},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, errs := testexecutor.New().CreateOperationContext(
				graphql.StartOperationTrace(context.Background()),
				&graphql.RawParams{Query: tc.query, Variables: tc.variables},
			)
			require.Len(t, errs, 1)
			assert.Equal(t, errcode.ValidationFailed, errs[0].Extensions["code"])
		})
	}
}

// Naming an operation the document does not contain is a validation failure
// too, and the message has to name what was asked for so the client can tell a
// typo from a deploy skew.
func TestCreateOperationContextRejectsAnUnknownOperationName(t *testing.T) {
	t.Parallel()

	_, errs := testexecutor.New().CreateOperationContext(
		graphql.StartOperationTrace(context.Background()),
		&graphql.RawParams{Query: "query Known { name }", OperationName: "Unknown"},
	)
	require.Len(t, errs, 1)
	assert.Equal(t, errcode.ValidationFailed, errs[0].Extensions["code"])
	assert.Contains(t, errs[0].Message, "Unknown")
}

// A parameter mutator can rewrite the query before it is parsed, which is how
// automatic persisted queries substitute a hash for a document.
func TestParameterMutatorCanRewriteTheQuery(t *testing.T) {
	t.Parallel()

	mutator := &paramMutator{
		MutateFunc: func(_ context.Context, params *graphql.RawParams) *gqlerror.Error {
			params.Query = "{ name }"
			return nil
		},
	}

	exec := testexecutor.New()
	exec.Use(mutator)

	opCtx := createOpCtx(t, exec, &graphql.RawParams{Query: "this would not parse"})

	assert.True(t, mutator.Invoked)
	assert.Equal(t, "{ name }", opCtx.RawQuery,
		"the rewritten query should be the one recorded and parsed")
}

// The operation context is installed on the context before the parameter
// mutators run, so a mutator can read the stats and settings already on it.
func TestParameterMutatorSeesTheOperationContext(t *testing.T) {
	t.Parallel()

	mutator := &paramMutator{}
	exec := testexecutor.New()
	exec.Use(mutator)

	createOpCtx(t, exec, &graphql.RawParams{Query: "{ name }"})

	require.True(t, mutator.Invoked)
	assert.True(t, mutator.SawOpCtx,
		"a parameter mutator should find the operation context on its context")
}

// A parameter mutator that rejects the request stops it before parsing, and its
// error is what the client sees. Swallowing it would run a query an extension
// had already refused.
func TestParameterMutatorCanRejectTheRequest(t *testing.T) {
	t.Parallel()

	refusal := gqlerror.Errorf("persisted query not found")
	mutator := &paramMutator{
		MutateFunc: func(context.Context, *graphql.RawParams) *gqlerror.Error {
			return refusal
		},
	}

	exec := testexecutor.New()
	exec.Use(mutator)

	opCtx, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()),
		&graphql.RawParams{Query: "{ name }"},
	)

	require.Len(t, errs, 1)
	assert.Equal(t, refusal.Message, errs[0].Message)
	require.NotNil(t, opCtx, "the operation context is returned even on refusal")
	// Parsing never happened, so there is no document to execute.
	assert.Nil(t, opCtx.Doc, "a refused request should not have been parsed")
}

// A context mutator runs after parsing and validation, so it can inspect the
// document, and its refusal also stops the request.
func TestContextMutatorRunsAndCanRejectTheRequest(t *testing.T) {
	t.Parallel()

	t.Run("it runs with a parsed document", func(t *testing.T) {
		t.Parallel()

		var sawDoc bool
		mutator := &ctxMutator{
			MutateFunc: func(_ context.Context, opCtx *graphql.OperationContext) *gqlerror.Error {
				sawDoc = opCtx.Doc != nil && opCtx.Operation != nil
				return nil
			},
		}
		exec := testexecutor.New()
		exec.Use(mutator)

		createOpCtx(t, exec, &graphql.RawParams{Query: "{ name }"})

		require.True(t, mutator.Invoked)
		assert.True(t, sawDoc,
			"a context mutator runs after parsing, so the document is available")
	})

	t.Run("its refusal stops the request", func(t *testing.T) {
		t.Parallel()

		refusal := gqlerror.Errorf("complexity limit exceeded")
		mutator := &ctxMutator{
			MutateFunc: func(context.Context, *graphql.OperationContext) *gqlerror.Error {
				return refusal
			},
		}
		exec := testexecutor.New()
		exec.Use(mutator)

		_, errs := exec.CreateOperationContext(
			graphql.StartOperationTrace(context.Background()),
			&graphql.RawParams{Query: "{ name }"},
		)

		require.Len(t, errs, 1)
		assert.Equal(t, refusal.Message, errs[0].Message)
	})
}

// A context mutator never runs for a request that failed earlier, because there
// is no validated operation context to hand it. An extension that assumed
// otherwise would act on a half-built request.
func TestContextMutatorIsSkippedWhenValidationFails(t *testing.T) {
	t.Parallel()

	mutator := &ctxMutator{}
	exec := testexecutor.New()
	exec.Use(mutator)

	_, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()),
		&graphql.RawParams{Query: "{ noSuchField }"},
	)

	require.NotEmpty(t, errs)
	assert.False(t, mutator.Invoked,
		"a context mutator should not run for a request that failed validation")
}
