package introspection

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// buildSyntheticSchema returns a schema with n object types, each with 10 fields taking 3
// arguments, plus an interface and a union over a slice of them — roughly the shape that
// makes introspection expensive on a real API.
func buildSyntheticSchema(tb testing.TB, n int) *ast.Schema {
	tb.Helper()

	var sb strings.Builder
	sb.WriteString("interface Node { id: ID! }\n")
	sb.WriteString("type Query { root: T0! }\n")
	for i := range n {
		fmt.Fprintf(&sb, "type T%d implements Node {\n  id: ID!\n", i)
		for f := range 10 {
			fmt.Fprintf(&sb,
				"  f%d(a: String, b: Int = 3, c: TInput): T%d\n", f, (i+f)%n)
		}
		sb.WriteString("}\n")
	}
	sb.WriteString("input TInput { x: String, y: Int }\n")
	sb.WriteString("enum TEnum { A B C }\n")

	members := make([]string, 0, min(n, 20))
	for i := range min(n, 20) {
		members = append(members, fmt.Sprintf("T%d", i))
	}
	fmt.Fprintf(&sb, "union TUnion = %s\n", strings.Join(members, " | "))

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "bench", Input: sb.String()})
	require.NoError(tb, err)
	return schema
}

// walkIntrospection performs the traversal that the generated __Type resolvers perform for
// a full introspection query.
func walkIntrospection(s *ast.Schema) int {
	wrapped := &Schema{schema: s}
	n := 0
	for _, typ := range wrapped.Types() {
		n += len(typ.Fields(true))
		n += len(typ.Interfaces())
		n += len(typ.PossibleTypes())
		n += len(typ.EnumValues(true))
		n += len(typ.InputFields())
	}
	n += len(wrapped.Directives())
	return n
}

func BenchmarkIntrospectionTraversal(b *testing.B) {
	for _, size := range []int{50, 500, 2000} {
		b.Run(fmt.Sprintf("types=%d", size), func(b *testing.B) {
			schema := buildSyntheticSchema(b, size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				walkIntrospection(schema)
			}
		})
	}
}

// TestIntrospectionResolvesEachTypeOnce is the measurement that decides whether memoizing
// introspection field computation is worth anything.
//
// Memoization only pays if the same computation happens more than once. A full
// introspection traversal reaches each type exactly once, so there is nothing for an
// intra-query cache to hit.
func TestIntrospectionResolvesEachTypeOnce(t *testing.T) {
	schema := buildSyntheticSchema(t, 20)
	wrapped := &Schema{schema: schema}

	fieldsCalls := map[string]int{}
	for _, typ := range wrapped.Types() {
		name := typ.Name()
		require.NotNil(t, name)
		fieldsCalls[*name]++
		typ.Fields(true)
	}

	require.NotEmpty(t, fieldsCalls)
	for name, calls := range fieldsCalls {
		require.Equal(t, 1, calls, "type %s visited more than once in one traversal", name)
	}
}
