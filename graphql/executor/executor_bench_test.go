package executor_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
	"github.com/99designs/gqlgen/graphql/handler/lru"
)

// These benchmarks cover the two halves of serving one GraphQL request:
// CreateOperationContext, which parses and validates the query, and
// DispatchOperation, which runs it. The split matters because only the first
// half is avoidable: a cache hit in parseQuery returns the parsed document and
// skips validation with it, so the query cache is the largest per-request lever
// gqlgen offers. BenchmarkCreateOperationContext measures what it is worth.

// benchQuery returns a query selecting the `name` field n times under distinct
// aliases, which grows the selection set without changing what is resolved.
func benchQuery(n int) string {
	var sb strings.Builder
	sb.WriteString("query bench {")
	for i := range n {
		fmt.Fprintf(&sb, " a%d: name", i)
	}
	sb.WriteString(" }")
	return sb.String()
}

func benchParams(query string) *graphql.RawParams {
	now := graphql.Now()
	return &graphql.RawParams{
		Query:    query,
		ReadTime: graphql.TraceTiming{Start: now, End: now},
	}
}

// BenchmarkCreateOperationContext measures parsing and validating a query, with
// and without a query cache in front of it.
//
// The cached variant is the configuration a production server should run; the
// uncached one is the default a server gets if it never calls SetQueryCache.
// The gap between them is the cost of that omission.
func BenchmarkCreateOperationContext(b *testing.B) {
	for _, fields := range []int{1, 10, 100} {
		query := benchQuery(fields)

		b.Run(fmt.Sprintf("fields=%d/cache=none", fields), func(b *testing.B) {
			exec := testexecutor.New()
			ctx := graphql.StartOperationTrace(context.Background())
			params := benchParams(query)

			b.ReportAllocs()
			for b.Loop() {
				_, errs := exec.CreateOperationContext(ctx, params)
				if len(errs) > 0 {
					b.Fatalf("unexpected errors: %v", errs)
				}
			}
		})

		b.Run(fmt.Sprintf("fields=%d/cache=lru", fields), func(b *testing.B) {
			exec := testexecutor.New()
			exec.SetQueryCache(lru.New[*ast.QueryDocument](128))
			ctx := graphql.StartOperationTrace(context.Background())
			params := benchParams(query)

			// Warm the cache outside the measured loop, so this measures the
			// hit path rather than one miss amortised over the run.
			_, errs := exec.CreateOperationContext(ctx, params)
			require.Empty(b, errs)

			b.ReportAllocs()
			for b.Loop() {
				_, errs := exec.CreateOperationContext(ctx, params)
				if len(errs) > 0 {
					b.Fatalf("unexpected errors: %v", errs)
				}
			}
		})
	}
}

// BenchmarkDispatchOperation measures executing an already parsed and validated
// operation, which is the half of a request no cache can remove.
//
// There is deliberately no selection-set size dimension here. testexecutor
// resolves one hardcoded field whatever the query selects, so a "fields=100"
// variant would report a number that has nothing to do with resolving 100
// fields. What this does measure is the dispatch pipeline around a resolve:
// the operation and response middleware chains, the response context and the
// marshalling. Measuring resolver scaling needs generated code, which
// codegen/testserver/benchmark already provides.
func BenchmarkDispatchOperation(b *testing.B) {
	exec := testexecutor.New()
	ctx := graphql.StartOperationTrace(context.Background())
	opCtx, errs := exec.CreateOperationContext(ctx, benchParams(benchQuery(1)))
	require.Empty(b, errs)

	b.ReportAllocs()
	for b.Loop() {
		handler, dispatchCtx := exec.DispatchOperation(ctx, opCtx)
		handler(dispatchCtx)
	}
}

// BenchmarkRequest measures both halves together with a warm query cache,
// which is what a correctly configured server spends per request.
//
// Measured, the two sizes come out the same: a cache hit makes per-request cost
// independent of how large the query is, because the only work left scales with
// what is resolved rather than with what was parsed. The small and large cases
// are both kept so that a change which reintroduces per-selection work on the
// hit path shows up as the two diverging.
func BenchmarkRequest(b *testing.B) {
	for _, fields := range []int{1, 100} {
		b.Run(fmt.Sprintf("fields=%d", fields), func(b *testing.B) {
			exec := testexecutor.New()
			exec.SetQueryCache(lru.New[*ast.QueryDocument](128))
			params := benchParams(benchQuery(fields))

			b.ReportAllocs()
			for b.Loop() {
				ctx := graphql.StartOperationTrace(context.Background())
				opCtx, errs := exec.CreateOperationContext(ctx, params)
				if len(errs) > 0 {
					b.Fatalf("unexpected errors: %v", errs)
				}
				handler, dispatchCtx := exec.DispatchOperation(ctx, opCtx)
				handler(dispatchCtx)
			}
		})
	}
}

// BenchmarkValidationFailure measures the whole path a request takes when it
// fails validation: the parse, the failed validation and the error response.
// This is the path a malformed or hostile query exercises, so it is the one an
// attacker can drive hardest, and no query cache shortens it.
func BenchmarkValidationFailure(b *testing.B) {
	exec := testexecutor.New()
	ctx := graphql.StartOperationTrace(context.Background())
	params := benchParams("{ noSuchFieldAnywhere }")

	_, errs := exec.CreateOperationContext(ctx, params)
	require.NotEmpty(b, errs, "the fixture query should fail validation")

	b.ReportAllocs()
	for b.Loop() {
		_, errs := exec.CreateOperationContext(ctx, params)
		exec.DispatchError(ctx, errs)
	}
}
