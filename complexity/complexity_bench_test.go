package complexity

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator/rules"

	"github.com/99designs/gqlgen/graphql"
)

// benchSchema returns the default complexity for every field. It does not record calls the way
// ExecutableSchemaMock does, so the benchmark measures Calculate and not a growing call log.
type benchSchema struct{}

func (benchSchema) Schema() *ast.Schema { return schema }

func (benchSchema) Complexity(
	context.Context,
	string, string,
	int,
	map[string]any,
) (int, bool) {
	return 0, false
}

func (benchSchema) Exec(context.Context) graphql.ResponseHandler { return nil }

// benchFragmentQuery returns a query whose fragment F<n> spreads F<n-1> twice, so F0 is
// reached 2^depth times. A depth of 0 spreads no fragment at all.
func benchFragmentQuery(depth int) string {
	if depth == 0 {
		return "{ object { scalar name list { scalar } } }"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "{ object { ...F%d } }\n", depth)
	b.WriteString("fragment F0 on Item { scalar }\n")
	for i := 1; i <= depth; i++ {
		fmt.Fprintf(&b, "fragment F%d on Item { ...F%d ...F%d }\n", i, i-1, i-1)
	}
	return b.String()
}

// BenchmarkCalculate measures Calculate on operations with nested fragment spreads. The
// cost should follow the size of the document, not the number of times a fragment is spread.
func BenchmarkCalculate(b *testing.B) {
	var es benchSchema
	for _, depth := range []int{0, 10, 20} {
		b.Run(fmt.Sprintf("fragment_depth=%d", depth), func(b *testing.B) {
			query := gqlparser.MustLoadQueryWithRules(
				schema,
				benchFragmentQuery(depth),
				rules.NewDefaultRules(),
			)
			op := query.Operations[0]

			b.ReportAllocs()
			for b.Loop() {
				Calculate(context.Background(), es, op, nil)
			}
		})
	}
}
