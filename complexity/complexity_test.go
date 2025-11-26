package complexity

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator/rules"

	"github.com/99designs/gqlgen/graphql"
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

func requireComplexity(t *testing.T, source string, complexity int, opts ...Option) {
	t.Helper()
	query := gqlparser.MustLoadQueryWithRules(schema, source, rules.NewDefaultRules())

	es := &graphql.ExecutableSchemaMock{
		ComplexityFunc: func(ctx context.Context, typeName, field string, childComplexity int, args map[string]any) (int, bool) {
			switch typeName + "." + field {
			case "ExpensiveItem.name":
				return 5, true
			case "Query.list", "Item.list":
				return int(args["size"].(int64)) * childComplexity, true
			case "Query.customObject":
				return 1, true
			}
			return 0, false
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	}

	actualComplexity := Calculate(context.TODO(), es, query.Operations[0], nil, opts...)
	require.Equal(t, complexity, actualComplexity)
}

func TestCalculate(t *testing.T) {
	t.Run("uses default cost", func(t *testing.T) {
		const query = `
		{
			scalar
		}
		`
		requireComplexity(t, query, 1)
	})

	t.Run("adds together fields", func(t *testing.T) {
		const query = `
		{
			scalar1: scalar
			scalar2: scalar
		}
		`
		requireComplexity(t, query, 2)
	})

	t.Run("a level of nesting adds complexity", func(t *testing.T) {
		const query = `
		{
			object {
				scalar
			}
		}
		`
		requireComplexity(t, query, 2)
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
		requireComplexity(t, query, 3)
	})

	t.Run("adds inline fragments", func(t *testing.T) {
		const query = `
		{
			... {
				scalar
			}
		}
		`
		requireComplexity(t, query, 1)
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
		requireComplexity(t, query, 1)
	})

	t.Run("uses custom complexity", func(t *testing.T) {
		const query = `
		{
			list {
				scalar
			}
		}
		`
		requireComplexity(t, query, 10)
	})

	t.Run("ignores negative custom complexity values", func(t *testing.T) {
		const query = `
		{
			list(size: -100) {
				scalar
			}
		}
		`
		requireComplexity(t, query, 2)
	})

	t.Run("interfaces take max concrete cost", func(t *testing.T) {
		const query = `
		{
			interface {
				name
			}
		}
		`
		requireComplexity(t, query, 6)
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
		requireComplexity(t, query, math.MaxInt64)
	})

	t.Run("fixed scalar value", func(t *testing.T) {
		const query = `
		{
			scalar
			object {
				scalar
				name
				list(size: 10) {
					scalar
				}
			}
		}
		`
		// object = 1
		// list = 1 (each scalar in the list is worth 0, hence 0*10=0,
		// but when custom complexity is less than 1 the calculation uses the default field value, i.e. 1)
		requireComplexity(t, query, 2, WithFixedScalarValue(0))
		// scalar = 2
		// object = 1
		// object.scalar = 2
		// object.name = 2
		// list = 2*10
		requireComplexity(t, query, 27, WithFixedScalarValue(2))
	})

	t.Run("ignore specified", func(t *testing.T) {
		const query = `
		{
			scalar
			object {
				scalar
				name
				list(size: 10) {
					scalar
				}
			}
		}
		`
		ignore := map[string]struct{}{
			"Query.scalar": {},
			"Item.name":    {},
		}
		requireComplexity(t, query, 12, WithIgnoreFields(ignore))
	})
}
