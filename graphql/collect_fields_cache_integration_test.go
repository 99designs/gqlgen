package graphql

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator/rules"
)

const collectFieldsSchemaSDL = `
    interface Node {
        name: String!
        email: String
    }

    type User implements Node {
        name: String!
        email: String
    }

    type Admin implements Node {
        name: String!
        email: String
        secret: String
    }

    type Query {
        search: [Node!]!
        user: User!
    }
`

var collectFieldsSchema = gqlparser.MustLoadSchema(&ast.Source{
	Name:  "collectFieldsCache",
	Input: collectFieldsSchemaSDL,
})

func TestCollectFieldsCache_InterfaceResult(t *testing.T) {
	const query = `
        query {
            search {
                name
                ... on User {
                    email
                }
                ... on Admin {
                    secret
                }
            }
        }
    `

	doc := gqlparser.MustLoadQueryWithRules(collectFieldsSchema, query, rules.NewDefaultRules())
	op := doc.Operations[0]
	searchField := op.SelectionSet[0].(*ast.Field)

	opCtx := &OperationContext{
		RawQuery:  query,
		Variables: nil,
		Doc:       doc,
		Operation: op,
	}

	userFields := CollectFields(opCtx, searchField.SelectionSet, []string{"User"})
	adminFields := CollectFields(opCtx, searchField.SelectionSet, []string{"Admin"})

	require.ElementsMatch(t, []string{"name", "email"}, fieldNames(userFields))
	require.ElementsMatch(t, []string{"name", "secret"}, fieldNames(adminFields))
	require.Equal(t, 2, opCtx.collectFieldsCache.Len())
}

func TestCollectFieldsCache_AliasResult(t *testing.T) {
	const query = `
        query {
            user1: user { name }
            user2: user { email }
        }
    `

	doc := gqlparser.MustLoadQueryWithRules(collectFieldsSchema, query, rules.NewDefaultRules())
	op := doc.Operations[0]

	opCtx := &OperationContext{
		RawQuery:  query,
		Variables: nil,
		Doc:       doc,
		Operation: op,
	}

	fields := CollectFields(opCtx, op.SelectionSet, nil)
	require.Equal(t, 1, opCtx.collectFieldsCache.Len())
	require.Len(t, fields, 2)

	expected := []struct {
		alias        string
		expectedName string
	}{
		{alias: "user1", expectedName: "name"},
		{alias: "user2", expectedName: "email"},
	}

	for _, check := range expected {
		var matched bool
		for _, f := range fields {
			alias := f.Alias
			if alias == "" {
				alias = f.Name
			}
			if alias == check.alias {
				require.Equal(t, []string{check.expectedName}, selectionNames(f.Selections))
				matched = true
				break
			}
		}
		require.True(t, matched, "expected alias %q not found", check.alias)
	}
}

func TestCollectFieldsCache_DirectiveResult(t *testing.T) {
	const query = `
        query Test($includeEmail: Boolean!, $skipName: Boolean!) {
            search {
                ... on User {
                    name @skip(if: $skipName)
                    email @include(if: $includeEmail)
                }
            }
        }
    `

	run := func(vars map[string]any) ([]CollectedField, int) {
		doc := gqlparser.MustLoadQueryWithRules(collectFieldsSchema, query, rules.NewDefaultRules())
		op := doc.Operations[0]
		searchField := op.SelectionSet[0].(*ast.Field)

		opCtx := &OperationContext{
			RawQuery:  query,
			Variables: vars,
			Doc:       doc,
			Operation: op,
		}

		first := CollectFields(opCtx, searchField.SelectionSet, []string{"User"})
		second := CollectFields(opCtx, searchField.SelectionSet, []string{"User"})
		require.Equal(t, first, second)
		return first, opCtx.collectFieldsCache.Len()
	}

	fieldsA, cacheA := run(map[string]any{"includeEmail": false, "skipName": false})
	require.Equal(t, 1, cacheA)
	require.Equal(t, []string{"name"}, fieldNames(fieldsA))

	fieldsB, cacheB := run(map[string]any{"includeEmail": true, "skipName": true})
	require.Equal(t, 1, cacheB)
	require.Equal(t, []string{"email"}, fieldNames(fieldsB))

	fieldsC, cacheC := run(map[string]any{"includeEmail": false, "skipName": true})
	require.Equal(t, 1, cacheC)
	require.Empty(t, fieldsC)

	fieldsD, cacheD := run(map[string]any{"includeEmail": true, "skipName": false})
	require.Equal(t, 1, cacheD)
	require.ElementsMatch(t, []string{"name", "email"}, fieldNames(fieldsD))
}

func fieldNames(fields []CollectedField) []string {
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.Name)
	}
	return names
}

func selectionNames(sel ast.SelectionSet) []string {
	names := make([]string, 0, len(sel))
	for _, s := range sel {
		if f, ok := s.(*ast.Field); ok {
			names = append(names, f.Name)
		}
	}
	return names
}
