package executor_test

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
)

func TestExecutor(t *testing.T) {
	exec := testexecutor.New()

	t.Run("calls query on executable schema", func(t *testing.T) {
		resp := query(exec, "", "{name}")
		assert.Equal(t, `{"name":"test"}`, string(resp.Data))
	})

	t.Run("invokes operation middleware in order", func(t *testing.T) {
		extlist := executor.Extensions(exec.Schema())

		var calls []string
		extlist.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
			calls = append(calls, "first")
			return next(ctx)
		})
		extlist.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
			calls = append(calls, "second")
			return next(ctx)
		})
		exec.SetExtensions(extlist.Extensions())

		resp := query(exec, "", "{name}")
		assert.Equal(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes response middleware in order", func(t *testing.T) {
		extlist := executor.Extensions(exec.Schema())

		var calls []string
		extlist.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
			calls = append(calls, "first")
			return next(ctx)
		})
		extlist.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
			calls = append(calls, "second")
			return next(ctx)
		})
		exec.SetExtensions(extlist.Extensions())

		resp := query(exec, "", "{name}")
		assert.Equal(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes field middleware in order", func(t *testing.T) {
		extlist := executor.Extensions(exec.Schema())

		var calls []string
		extlist.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			calls = append(calls, "first")
			return next(ctx)
		})
		extlist.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			calls = append(calls, "second")
			return next(ctx)
		})
		exec.SetExtensions(extlist.Extensions())

		resp := query(exec, "", "{name}")
		assert.Equal(t, `{"name":"test"}`, string(resp.Data))
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("get query parse error in AroundResponses", func(t *testing.T) {
		extlist := executor.Extensions(exec.Schema())

		var errors1 gqlerror.List
		var errors2 gqlerror.List
		extlist.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
			resp := next(ctx)
			errors1 = graphql.GetErrors(ctx)
			errors2 = resp.Errors
			return resp
		})
		exec.SetExtensions(extlist.Extensions())

		resp := query(exec, "", "invalid")
		assert.Equal(t, "", string(resp.Data))
		assert.Equal(t, 1, len(resp.Errors))
		assert.Equal(t, 1, len(errors1))
		assert.Equal(t, 1, len(errors2))
	})

	t.Run("query caching", func(t *testing.T) {
		ctx := context.Background()
		cache := &graphql.MapCache{}
		exec.SetQueryCache(cache)
		qry := `query Foo {name}`

		t.Run("cache miss populates cache", func(t *testing.T) {
			resp := query(exec, "Foo", qry)
			assert.Equal(t, `{"name":"test"}`, string(resp.Data))

			cacheDoc, ok := cache.Get(ctx, qry)
			require.True(t, ok)
			require.Equal(t, "Foo", cacheDoc.(*ast.QueryDocument).Operations[0].Name)
		})

		t.Run("cache hits use document from cache", func(t *testing.T) {
			doc, err := parser.ParseQuery(&ast.Source{Input: `query Bar {name}`})
			require.Nil(t, err)
			cache.Add(ctx, qry, doc)

			resp := query(exec, "Bar", qry)
			assert.Equal(t, `{"name":"test"}`, string(resp.Data))

			cacheDoc, ok := cache.Get(ctx, qry)
			require.True(t, ok)
			require.Equal(t, "Bar", cacheDoc.(*ast.QueryDocument).Operations[0].Name)
		})
	})
}

func TestErrorServer(t *testing.T) {
	exec := testexecutor.NewError()

	t.Run("get resolver error in AroundResponses", func(t *testing.T) {
		extlist := executor.Extensions(exec.Schema())

		var errors1 gqlerror.List
		var errors2 gqlerror.List
		extlist.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
			resp := next(ctx)
			errors1 = graphql.GetErrors(ctx)
			errors2 = resp.Errors
			return resp
		})
		exec.SetExtensions(extlist.Extensions())

		resp := query(exec, "", "{name}")
		assert.Equal(t, "null", string(resp.Data))
		assert.Equal(t, 1, len(errors1))
		assert.Equal(t, 1, len(errors2))
	})
}

func query(exec *testexecutor.TestExecutor, op, q string) *graphql.Response {
	return exec.Exec(context.Background(), &graphql.RawParams{
		Query:         q,
		OperationName: op,
	})
}
