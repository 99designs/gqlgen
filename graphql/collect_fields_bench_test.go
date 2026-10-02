package graphql

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// CollectFields runs for every object the executor resolves, so its cost is
// multiplied by the shape of the response rather than paid once per request.
// It is also memoized per operation, and these benchmarks are what decide
// whether that memoization earns its complexity: the cold variant pays
// collectFields, the warm one pays only the cache key and a map lookup.

// benchSelectionSet builds a query selecting n fields, half of them behind
// fragments, and returns the selection set of its operation along with an
// OperationContext to collect against.
func benchSelectionSet(tb testing.TB, n int) (*OperationContext, ast.SelectionSet) {
	tb.Helper()

	schema := gqlparser.MustLoadSchema(&ast.Source{Name: "bench", Input: `
		interface Node { id: ID! }
		type Query { root: Thing! }
		type Thing implements Node {
			id: ID!
			a: String
			b: String
			c: String
		}
	`})

	var sb strings.Builder
	sb.WriteString("query bench { root {")
	for i := range n {
		// Alternate plain fields with inline fragments, which is the shape that
		// makes collectFields recurse and dedupe rather than walk a flat list.
		if i%2 == 0 {
			fmt.Fprintf(&sb, " f%d: a", i)
			continue
		}
		fmt.Fprintf(&sb, " ... on Thing { g%d: b }", i)
	}
	sb.WriteString(" } }")

	doc, err := parser.ParseQuery(&ast.Source{Name: "bench", Input: sb.String()})
	require.NoError(tb, err)
	require.NotEmpty(tb, doc.Operations)

	opCtx := &OperationContext{
		RawQuery:  sb.String(),
		Doc:       doc,
		Operation: doc.Operations[0],
	}
	// The operation selects a single root field; its selection set is the one
	// worth collecting repeatedly.
	root, ok := doc.Operations[0].SelectionSet[0].(*ast.Field)
	require.True(tb, ok, "the fixture query should select a field at the root")
	require.NotNil(tb, schema)
	return opCtx, root.SelectionSet
}

// BenchmarkCollectFields measures field collection on a cold operation context,
// where every call does the work, against a warm one, where the per-operation
// cache answers. A resolver walking a list of 100 objects makes 100 of these
// calls against the same selection set, so the warm number is the one that
// describes a real response.
func BenchmarkCollectFields(b *testing.B) {
	satisfies := []string{"Thing", "Node"}

	for _, fields := range []int{4, 40, 200} {
		b.Run(fmt.Sprintf("fields=%d/cache=cold", fields), func(b *testing.B) {
			_, selSet := benchSelectionSet(b, fields)

			b.ReportAllocs()
			for b.Loop() {
				// A fresh context per iteration, so every call misses.
				opCtx := &OperationContext{}
				CollectFields(opCtx, selSet, satisfies)
			}
		})

		b.Run(fmt.Sprintf("fields=%d/cache=warm", fields), func(b *testing.B) {
			opCtx, selSet := benchSelectionSet(b, fields)
			// Warm the cache outside the loop so this measures the hit path.
			require.NotEmpty(b, CollectFields(opCtx, selSet, satisfies))
			require.Equal(b, 1, opCtx.collectFieldsCache.Len())

			b.ReportAllocs()
			for b.Loop() {
				CollectFields(opCtx, selSet, satisfies)
			}
		})
	}
}

// BenchmarkCollectFieldsCacheKey isolates the cost paid on every call, hit or
// miss. It bounds how cheap the warm path can ever be, so a change that makes
// the key more expensive caps the memoization's benefit whatever else improves.
func BenchmarkCollectFieldsCacheKey(b *testing.B) {
	_, selSet := benchSelectionSet(b, 40)
	satisfies := []string{"Thing", "Node"}

	b.ReportAllocs()
	for b.Loop() {
		makeCollectFieldsCacheKey(selSet, satisfies)
	}
}
