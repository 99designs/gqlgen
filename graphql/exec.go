package graphql

import (
	"context"
	"fmt"

	"github.com/vektah/gqlparser/ast"
)

type ExecutableSchema interface {
	Schema() *ast.Schema

	Query(ctx context.Context, op *ast.OperationDefinition) *Response
	Mutation(ctx context.Context, op *ast.OperationDefinition) *Response
	Subscription(ctx context.Context, op *ast.OperationDefinition) func() *Response
}

func CollectFields(doc *ast.QueryDocument, selSet ast.SelectionSet, satisfies []string, variables map[string]interface{}) []CollectedField {
	return collectFields(doc, selSet, satisfies, variables, map[string]bool{})
}

func collectFields(doc *ast.QueryDocument, selSet ast.SelectionSet, satisfies []string, variables map[string]interface{}, visited map[string]bool) []CollectedField {
	var groupedFields []CollectedField

	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *ast.Field:
			f := getOrCreateField(&groupedFields, sel.Alias, func() CollectedField {
				f := CollectedField{
					Alias: sel.Alias,
					Name:  sel.Name,
				}
				if len(sel.Arguments) > 0 {
					f.Args = map[string]interface{}{}
					for _, arg := range sel.Arguments {
						if arg.Value.Kind == ast.Variable {
							if val, ok := variables[arg.Value.Raw]; ok {
								f.Args[arg.Name] = val
							}
						} else {
							var err error
							f.Args[arg.Name], err = arg.Value.Value(variables)
							if err != nil {
								panic(err)
							}
						}
					}
				}
				return f
			})

			f.Selections = append(f.Selections, sel.SelectionSet...)
		case *ast.InlineFragment:
			if !instanceOf(sel.TypeCondition, satisfies) {
				continue
			}

			for _, childField := range collectFields(doc, sel.SelectionSet, satisfies, variables, visited) {
				f := getOrCreateField(&groupedFields, childField.Name, func() CollectedField { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		case *ast.FragmentSpread:
			fragmentName := sel.Name
			if _, seen := visited[fragmentName]; seen {
				continue
			}
			visited[fragmentName] = true

			fragment := doc.Fragments.ForName(fragmentName)
			if fragment == nil {
				// should never happen, validator has already run
				panic(fmt.Errorf("missing fragment %s", fragmentName))
			}

			if !instanceOf(fragment.TypeCondition, satisfies) {
				continue
			}

			for _, childField := range collectFields(doc, fragment.SelectionSet, satisfies, variables, visited) {
				f := getOrCreateField(&groupedFields, childField.Name, func() CollectedField { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
}

type CollectedField struct {
	Alias      string
	Name       string
	Args       map[string]interface{}
	Selections ast.SelectionSet
}

func instanceOf(val string, satisfies []string) bool {
	for _, s := range satisfies {
		if val == s {
			return true
		}
	}
	return false
}

func getOrCreateField(c *[]CollectedField, name string, creator func() CollectedField) *CollectedField {
	for i, cf := range *c {
		if cf.Alias == name {
			return &(*c)[i]
		}
	}

	f := creator()

	*c = append(*c, f)
	return &(*c)[len(*c)-1]
}
