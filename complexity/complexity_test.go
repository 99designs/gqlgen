package complexity

import (
	"context"
	"math"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

var schema = gqlparser.MustLoadSchema(
	&ast.Source{
		Name: "test.graphql",
		Input: `
		interface NameInterface {
			name: String
		}

		type Item implements NameInterface {
			scalar: String
			name: String
			list(size: Int = 10): [Item]
		}

		type ExpensiveItem implements NameInterface {
			name: String
		}

		type Named {
			name: String
		}

		union NameUnion = Item | Named

		type Query {
			scalar: String
			object: Item
			interface: NameInterface
			union: NameUnion
			customObject: Item
			list(size: Int = 10): [Item]
		}
		`,
	},
)

func requireComplexity(t *testing.T, source string, vars map[string]interface{}, complexity int) {
	t.Helper()
	query := gqlparser.MustLoadQuery(schema, source)
	es := &executableSchemaStub{}
	actualComplexity := Calculate(es, query.Operations[0], vars)
	require.Equal(t, complexity, actualComplexity)
}

func TestCalculate(t *testing.T) {
	t.Run("uses default cost", func(t *testing.T) {
		const query = `
		{
			scalar
		}
		`
		requireComplexity(t, query, nil, 1)
	})

	t.Run("adds together fields", func(t *testing.T) {
		const query = `
		{
			scalar1: scalar
			scalar2: scalar
		}
		`
		requireComplexity(t, query, nil, 2)
	})

	t.Run("a level of nesting adds complexity", func(t *testing.T) {
		const query = `
		{
			object {
				scalar
			}
		}
		`
		requireComplexity(t, query, nil, 2)
	})

	t.Run("adds together children", func(t *testing.T) {
		const query = `
		{
			scalar
			object {
				scalar
			}
		}
		`
		requireComplexity(t, query, nil, 3)
	})

	t.Run("adds inline fragments", func(t *testing.T) {
		const query = `
		{
			... {
				scalar
			}
		}
		`
		requireComplexity(t, query, nil, 1)
	})

	t.Run("adds fragments", func(t *testing.T) {
		const query = `
		{
			... Fragment
		}

		fragment Fragment on Query {
			scalar
		}
		`
		requireComplexity(t, query, nil, 1)
	})

	t.Run("uses custom complexity", func(t *testing.T) {
		const query = `
		{
			list {
				scalar
			}
		}
		`
		requireComplexity(t, query, nil, 10)
	})

	t.Run("ignores negative custom complexity values", func(t *testing.T) {
		const query = `
		{
			list(size: -100) {
				scalar
			}
		}
		`
		requireComplexity(t, query, nil, 2)
	})

	t.Run("custom complexity must be >= child complexity", func(t *testing.T) {
		const query = `
		{
			customObject {
				list(size: 100) {
					scalar
				}
			}
		}
		`
		requireComplexity(t, query, nil, 101)
	})

	t.Run("interfaces take max concrete cost", func(t *testing.T) {
		const query = `
		{
			interface {
				name
			}
		}
		`
		requireComplexity(t, query, nil, 6)
	})

	t.Run("guards against integer overflow", func(t *testing.T) {
		if maxInt == math.MaxInt32 {
			// this test is written assuming 64-bit ints
			t.Skip()
		}
		const query = `
		{
			list1: list(size: 2147483647) {
				list(size: 2147483647) {
					list(size: 2) {
						scalar
					}
				}
			}
			# total cost so far: 2*0x7fffffff*0x7fffffff
			# = 0x7ffffffe00000002
			# Adding the same again should cause overflow
			list2: list(size: 2147483647) {
				list(size: 2147483647) {
					list(size: 2) {
						scalar
					}
				}
			}
		}
		`
		requireComplexity(t, query, nil, math.MaxInt64)
	})
}

type executableSchemaStub struct {
}

var _ graphql.ExecutableSchema = &executableSchemaStub{}

func (e *executableSchemaStub) Schema() *ast.Schema {
	return schema
}

func (e *executableSchemaStub) Complexity(typeName, field string, childComplexity int, args map[string]interface{}) (int, bool) {
	switch typeName + "." + field {
	case "ExpensiveItem.name":
		return 5, true
	case "Query.list", "Item.list":
		return int(args["size"].(int64)) * childComplexity, true
	case "Query.customObject":
		return 1, true
	}
	return 0, false
}

func (e *executableSchemaStub) Query(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
	panic("Query should never be called by complexity calculations")
}

func (e *executableSchemaStub) Mutation(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
	panic("Mutation should never be called by complexity calculations")
}

func (e *executableSchemaStub) Subscription(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
	panic("Subscription should never be called by complexity calculations")
}
