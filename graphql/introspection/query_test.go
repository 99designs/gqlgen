package introspection

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
	"github.com/vektah/gqlparser/v2/validator/rules"
)

// Each query must ask for everything gqlparser's prelude lets a client
// introspect, apart from the fields it leaves out on purpose. A gqlparser upgrade
// that adds an introspection field therefore fails here until gqlgen requests
// it or makes it a QueryOptions field.
func TestQueryRequestsEveryIntrospectionField(t *testing.T) {
	cases := map[string]struct {
		query   string
		omitted []string
	}{
		"Query": {query: Query, omitted: []string{"__Type.isOneOf"}},
		"NewQuery zero value": {
			query:   NewQuery(QueryOptions{}),
			omitted: []string{"__Type.isOneOf"},
		},
		"NewQuery with OneOf": {query: NewQuery(QueryOptions{OneOf: true})},
	}
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: "type Query { f: Int }"})
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			doc := gqlparser.MustLoadQueryWithRules(schema, tc.query, rules.NewDefaultRules())

			selected := map[string]*ast.Field{}
			var collect func(ast.SelectionSet)
			collect = func(set ast.SelectionSet) {
				for _, sel := range set {
					switch sel := sel.(type) {
					case *ast.Field:
						selected[sel.ObjectDefinition.Name+"."+sel.Name] = sel
						collect(sel.SelectionSet)
					case *ast.InlineFragment:
						collect(sel.SelectionSet)
					case *ast.FragmentSpread:
						collect(sel.Definition.SelectionSet)
					}
				}
			}
			for _, op := range doc.Operations {
				collect(op.SelectionSet)
			}

			for _, def := range validator.PreludeDocument().Definitions {
				if def.Kind != ast.Object || !strings.HasPrefix(def.Name, "__") {
					continue
				}
				for _, fieldDef := range def.Fields {
					field := def.Name + "." + fieldDef.Name
					if slices.Contains(tc.omitted, field) {
						require.NotContains(t, selected, field, "%s should omit %s", name, field)
						continue
					}
					require.Contains(t, selected, field, "%s does not request %s", name, field)

					if fieldDef.Arguments.ForName("includeDeprecated") != nil {
						arg := selected[field].Arguments.ForName("includeDeprecated")
						require.NotNil(t, arg, "%s omits %s(includeDeprecated:)", name, field)
						require.Equal(t, "true", arg.Value.Raw)
					}
				}
			}
		})
	}

	require.Equal(t, Query, NewQuery(QueryOptions{}))
}

// TypeRef must nest ofType as deep as graphql-js's default typeDepth of 9.
// Otherwise a deeply wrapped type such as [[[[Int!]!]!]!], which needs 8
// levels, comes back without its innermost type.
func TestQueryTypeRefDepth(t *testing.T) {
	cases := map[string]string{
		"Query":               Query,
		"NewQuery with OneOf": NewQuery(QueryOptions{OneOf: true}),
	}
	schema := gqlparser.MustLoadSchema(&ast.Source{Input: "type Query { f: Int }"})
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			doc := gqlparser.MustLoadQueryWithRules(schema, query, rules.NewDefaultRules())

			depth := 0
			set := doc.Fragments.ForName("TypeRef").SelectionSet
			for {
				var next ast.SelectionSet
				for _, sel := range set {
					if field, ok := sel.(*ast.Field); ok && field.Name == "ofType" {
						next = field.SelectionSet
					}
				}
				if next == nil {
					break
				}
				depth++
				set = next
			}
			require.Equal(t, 9, depth)
		})
	}
}
