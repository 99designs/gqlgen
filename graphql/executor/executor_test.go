package executor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
)

func TestExecutor(t *testing.T) {
	exec := testexecutor.New()

	t.Run("calls query on executable schema", func(t *testing.T) {
		resp, _ := query(exec, "", "{name}")
		assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
	})

	t.Run("validates operation", func(t *testing.T) {
		t.Run("no operation", func(t *testing.T) {
			resp, _ := query(exec, "", "")
			assert.Empty(t, string(resp.Data))
			assert.Len(t, resp.Errors, 1)
			assert.Equal(t, errcode.ValidationFailed, resp.Errors[0].Extensions["code"])
		})

		t.Run("bad operation", func(t *testing.T) {
			resp, _ := query(exec, "badOp", "query test { name }")
			assert.Empty(t, string(resp.Data))
			assert.Len(t, resp.Errors, 1)
			assert.Equal(t, errcode.ValidationFailed, resp.Errors[0].Extensions["code"])
		})

		t.Run("invalid variables", func(t *testing.T) {
			resp, oc := query(exec, "", "query test($id: Int!) {find(id: $id)}", variable("id", "invalid"))
			assert.Empty(t, string(resp.Data))
			assert.Len(t, resp.Errors, 1)
			assert.Equal(t, errcode.ValidationFailed, resp.Errors[0].Extensions["code"])
			assert.NotNil(t, oc)
			assert.Equal(t, "invalid", oc.Variables["id"])
		})
	})

	t.Run("invokes operation middleware in order", func(t *testing.T) {
		var calls []string
		exec.AroundOperations(
			func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
				calls = append(calls, "first")
				return next(ctx)
			},
		)
		exec.AroundOperations(
			func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
				calls = append(calls, "second")
				return next(ctx)
			},
		)

		resp, _ := query(exec, "", "{name}")
		assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes response middleware in order", func(t *testing.T) {
		var calls []string
		exec.AroundResponses(
			func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
				calls = append(calls, "first")
				return next(ctx)
			},
		)
		exec.AroundResponses(
			func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
				calls = append(calls, "second")
				return next(ctx)
			},
		)

		resp, _ := query(exec, "", "{name}")
		assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes root field middleware in order", func(t *testing.T) {
		var calls []string
		exec.AroundRootFields(
			func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
				calls = append(calls, "first")
				return next(ctx)
			},
		)
		exec.AroundRootFields(
			func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
				calls = append(calls, "second")
				return next(ctx)
			},
		)

		resp, _ := query(exec, "", "{name}")
		assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes field middleware in order", func(t *testing.T) {
		var calls []string
		exec.AroundFields(func(ctx context.Context, next graphql.Resolver) (res any, err error) {
			calls = append(calls, "first")
			return next(ctx)
		})
		exec.AroundFields(func(ctx context.Context, next graphql.Resolver) (res any, err error) {
			calls = append(calls, "second")
			return next(ctx)
		})

		resp, _ := query(exec, "", "{name}")
		assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes operation mutators", func(t *testing.T) {
		var calls []string
		exec.Use(&testParamMutator{
			Mutate: func(ctx context.Context, req *graphql.RawParams) *gqlerror.Error {
				calls = append(calls, "param")
				return nil
			},
		})
		exec.Use(&testCtxMutator{
			Mutate: func(ctx context.Context, opCtx *graphql.OperationContext) *gqlerror.Error {
				calls = append(calls, "context")
				return nil
			},
		})
		resp, _ := query(exec, "", "{name}")
		assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"param", "context"}, calls)
	})

	t.Run("get query parse error in AroundResponses", func(t *testing.T) {
		var errors1 gqlerror.List
		var errors2 gqlerror.List
		exec.AroundResponses(
			func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
				resp := next(ctx)
				errors1 = graphql.GetErrors(ctx)
				errors2 = resp.Errors
				return resp
			},
		)

		resp, _ := query(exec, "", "invalid")
		assert.Empty(t, string(resp.Data))
		assert.Len(t, resp.Errors, 1)
		assert.Len(t, errors1, 1)
		assert.Len(t, errors2, 1)
	})

	t.Run("query caching", func(t *testing.T) {
		ctx := context.Background()
		cache := &graphql.MapCache[*ast.QueryDocument]{}
		exec.SetQueryCache(cache)
		qry := `query Foo {name}`

		t.Run("cache miss populates cache", func(t *testing.T) {
			resp, _ := query(exec, "Foo", qry)
			assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))

			cacheDoc, ok := cache.Get(ctx, qry)
			require.True(t, ok)
			require.Equal(t, "Foo", cacheDoc.Operations[0].Name)
		})

		t.Run("cache hits use document from cache", func(t *testing.T) {
			doc, err := parser.ParseQuery(&ast.Source{Input: `query Bar {name}`})
			require.NoError(t, err)
			cache.Add(ctx, qry, doc)

			resp, _ := query(exec, "Bar", qry)
			assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))

			cacheDoc, ok := cache.Get(ctx, qry)
			require.True(t, ok)
			require.Equal(t, "Bar", cacheDoc.Operations[0].Name)
		})
	})
}

func TestExecutorDisableSuggestion(t *testing.T) {
	t.Run("by default, the error message will include suggestions", func(t *testing.T) {
		exec := testexecutor.New()
		resp, _ := query(exec, "", "{nam}")
		assert.Empty(t, string(resp.Data))
		assert.Equal(
			t,
			"input:1:2: Cannot query field \"nam\" on type \"Query\". Did you mean \"name\"?\n",
			resp.Errors.Error(),
		)
	})

	t.Run("disable suggestion, the error message will not include suggestions", func(t *testing.T) {
		exec := testexecutor.New()
		exec.SetDisableSuggestion(true)
		resp, _ := query(exec, "", "{nam}")
		assert.Empty(t, string(resp.Data))
		assert.Len(t, resp.Errors, 1)
		assert.Equal(
			t,
			"input:1:2: Cannot query field \"nam\" on type \"Query\".\n",
			resp.Errors.Error(),
		)

		// check if the error message is displayed correctly even if an error occurs multiple times
		resp, _ = query(exec, "", "{nam}")
		assert.Empty(t, string(resp.Data))
		assert.Len(t, resp.Errors, 1)
		assert.Equal(
			t,
			"input:1:2: Cannot query field \"nam\" on type \"Query\".\n",
			resp.Errors.Error(),
		)
	})
}

type testParamMutator struct {
	Mutate func(context.Context, *graphql.RawParams) *gqlerror.Error
}

func (m *testParamMutator) ExtensionName() string {
	return "Operation: Mutate Parameters"
}

func (m *testParamMutator) Validate(s graphql.ExecutableSchema) error {
	return nil
}

func (m *testParamMutator) MutateOperationParameters(
	ctx context.Context,
	r *graphql.RawParams,
) *gqlerror.Error {
	return m.Mutate(ctx, r)
}

type testCtxMutator struct {
	Mutate func(context.Context, *graphql.OperationContext) *gqlerror.Error
}

func (m *testCtxMutator) ExtensionName() string {
	return "Operation: Mutate the Context"
}

func (m *testCtxMutator) Validate(s graphql.ExecutableSchema) error {
	return nil
}

func (m *testCtxMutator) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	return m.Mutate(ctx, opCtx)
}

func TestErrorServer(t *testing.T) {
	exec := testexecutor.NewError()

	t.Run("get resolver error in AroundResponses", func(t *testing.T) {
		var errors1 gqlerror.List
		var errors2 gqlerror.List
		exec.AroundResponses(
			func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
				resp := next(ctx)
				errors1 = graphql.GetErrors(ctx)
				errors2 = resp.Errors
				return resp
			},
		)

		resp, _ := query(exec, "", "{name}")
		assert.Equal(t, "null", string(resp.Data))
		assert.Len(t, errors1, 1)
		assert.Len(t, errors2, 1)
	})
}

type paramOption func(*graphql.RawParams)

func variable(name string, v any) paramOption {
	return func(p *graphql.RawParams) {
		p.Variables[name] = v
	}
}

func query(exec *testexecutor.TestExecutor, op, q string, opts ...paramOption) (*graphql.Response, *graphql.OperationContext) {
	ctx := graphql.StartOperationTrace(context.Background())
	now := graphql.Now()
	params := &graphql.RawParams{
		Query:         q,
		OperationName: op,
		ReadTime: graphql.TraceTiming{
			Start: now,
			End:   now,
		},
	}
	for _, opt := range opts {
		opt(params)
	}
	rc, err := exec.CreateOperationContext(ctx, params)
	if err != nil {
		return exec.DispatchError(ctx, err), rc
	}
	resp, ctx2 := exec.DispatchOperation(ctx, rc)
	return resp(ctx2), rc
}
