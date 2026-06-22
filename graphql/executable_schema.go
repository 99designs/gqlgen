//go:generate go run github.com/matryer/moq -out executable_schema_mock.go . ExecutableSchema

package graphql

import (
	"context"
	"fmt"
	"slices"

	"github.com/vektah/gqlparser/v2/ast"
)

type ExecutableSchema interface {
	Schema() *ast.Schema

	Complexity(
		ctx context.Context,
		typeName, fieldName string,
		childComplexity int,
		args map[string]any,
	) (int, bool)
	Exec(ctx context.Context) ResponseHandler
}

// ExecutableSchemaWithEventContext is the optional interface a generated
// [ExecutableSchema] implements when at least one of its subscription fields
// is annotated with the @subscriptionContext directive. The graphql executor
// type-asserts for this interface; absence falls back to
// [ExecutableSchema.Exec] with no behavior change.
//
// Implementations must guarantee that the returned [ResponseHandlerWithContext]
// reports a context derived from the subscription's request context for every
// iteration — never an unrelated background context — so request-scoped
// values remain reachable.
type ExecutableSchemaWithEventContext interface {
	ExecutableSchema
	ExecWithEventContext(ctx context.Context) ResponseHandlerWithContext
}

// CollectFields returns the set of fields from an ast.SelectionSet where all collected fields
// satisfy at least one of the GraphQL types passed through satisfies. Providing an empty slice for
// satisfies will collect all fields regardless of fragment type conditions.
func CollectFields(
	reqCtx *OperationContext,
	selSet ast.SelectionSet,
	satisfies []string,
) []CollectedField {
	cacheKey := makeCollectFieldsCacheKey(selSet, satisfies)

	if cached, ok := reqCtx.collectFieldsCache.Get(cacheKey); ok {
		return cached
	}

	result := collectFields(reqCtx, selSet, satisfies, map[string]bool{}, false)

	return reqCtx.collectFieldsCache.Add(cacheKey, result)
}

func collectFields(
	reqCtx *OperationContext,
	selSet ast.SelectionSet,
	satisfies []string,
	visited map[string]bool,
	parentIsDeferredFragment bool,
) []CollectedField {
	groupedFields := make([]CollectedField, 0, len(selSet))

	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *ast.Field:
			if !shouldIncludeNode(sel.Directives, reqCtx.Variables) {
				continue
			}
			f := getOrCreateAndAppendField(
				&groupedFields,
				sel.Name,
				sel.Alias,
				sel.ObjectDefinition,
				func() CollectedField {
					return CollectedField{Field: sel}
				},
			)

			if !parentIsDeferredFragment {
				f.IsNonDeferrable = true
			}
			f.Selections = append(f.Selections, sel.SelectionSet...)

		case *ast.InlineFragment:
			if !shouldIncludeNode(sel.Directives, reqCtx.Variables) {
				continue
			}
			if !doesFragmentConditionMatch(sel.TypeCondition, satisfies) {
				continue
			}

			shouldDefer, label := deferrable(sel.Directives, reqCtx.Variables)
			childFields := collectFields(
				reqCtx,
				sel.SelectionSet,
				satisfies,
				visited,
				shouldDefer || parentIsDeferredFragment,
			)
			for _, childField := range childFields {
				var isChildField bool
				f := getOrCreateAndAppendField(
					&groupedFields, childField.Name, childField.Alias, childField.ObjectDefinition,
					func() CollectedField {
						isChildField = true
						return childField
					})

				if !isChildField {
					f.Selections = append(f.Selections, childField.Selections...)
					f.Deferrables = slices.Grow(f.Deferrables, len(childField.Deferrables)+1)
					f.Deferrables = append(f.Deferrables, childField.Deferrables...)
					f.IsNonDeferrable = f.IsNonDeferrable || childField.IsNonDeferrable
				}

				if shouldDefer {
					f.Deferrables = append(f.Deferrables, &Deferrable{
						Label: label,
					})
				}
			}

		case *ast.FragmentSpread:
			fragmentName := sel.Name
			if _, seen := visited[fragmentName]; seen {
				continue
			}
			if !shouldIncludeNode(sel.Directives, reqCtx.Variables) {
				continue
			}
			visited[fragmentName] = true

			fragment := reqCtx.Doc.Fragments.ForName(fragmentName)
			if fragment == nil {
				// should never happen, validator has already run
				panic(fmt.Errorf("missing fragment %s", fragmentName))
			}
			if !doesFragmentConditionMatch(fragment.TypeCondition, satisfies) {
				continue
			}

			shouldDefer, label := deferrable(sel.Directives, reqCtx.Variables)

			childFields := collectFields(
				reqCtx,
				fragment.SelectionSet,
				satisfies,
				visited,
				shouldDefer || parentIsDeferredFragment,
			)
			for _, childField := range childFields {
				var isChildField bool
				f := getOrCreateAndAppendField(&groupedFields,
					childField.Name, childField.Alias, childField.ObjectDefinition,
					func() CollectedField {
						isChildField = true
						return childField
					})

				if !isChildField {
					f.Selections = append(f.Selections, childField.Selections...)
					f.Deferrables = slices.Grow(f.Deferrables, len(childField.Deferrables)+1)
					f.Deferrables = append(f.Deferrables, childField.Deferrables...)
					f.IsNonDeferrable = f.IsNonDeferrable || childField.IsNonDeferrable
				}

				if shouldDefer {
					f.Deferrables = append(f.Deferrables, &Deferrable{
						Label: label,
					})
				}
			}

		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
}

type CollectedField struct {
	*ast.Field

	Selections ast.SelectionSet

	// IsNonDeferrable reports whether the field cannot be deferred,
	// regardless of what [Deferrables] reports. This is the case when the
	// same field is selected in a query from both within a deferred fragment,
	// and outside of one.
	//
	// Example - account cannot be deferred in this example:
	//	query {
	//		... @defer(label: "foo") {
	//			account {
	//				id
	//			}
	//		}
	//
	//		account {
	//			id
	//		}
	//	}
	IsNonDeferrable bool
	Deferrables     []*Deferrable
}

// IsDeferred reports whether this field's resolution should be deferred
// (collected into a [FieldSetView] keyed by every label in [Deferrables])
// rather than emitted in the initial response. A field is deferred when it
// appears inside at least one @defer fragment and is not also selected
// outside of one — the [IsNonDeferrable] flag overrides Deferrables.
func (f CollectedField) IsDeferred() bool {
	return len(f.Deferrables) > 0 && !f.IsNonDeferrable
}

func doesFragmentConditionMatch(typeCondition string, satisfies []string) bool {
	// To allow simplified "collect all" types behavior, pass an empty list of types
	// that the type condition must satisfy: we will apply the fragment regardless of
	// type condition.
	if len(satisfies) == 0 {
		return true
	}

	// When the type condition is not set (... { field }) we will apply the fragment
	// to any satisfying types.
	if typeCondition == "" {
		return true
	}

	// To handle abstract types we pass in a list of all known types that the current
	// type will satisfy.
	return slices.Contains(satisfies, typeCondition)
}

func getOrCreateAndAppendField(
	c *[]CollectedField,
	name, alias string,
	objectDefinition *ast.Definition,
	creator func() CollectedField,
) *CollectedField {
	for i, cf := range *c {
		if cf.Name == name && cf.Alias == alias {
			if cf.ObjectDefinition == objectDefinition {
				return &(*c)[i]
			}

			if cf.ObjectDefinition == nil || objectDefinition == nil {
				continue
			}

			if cf.ObjectDefinition.Name == objectDefinition.Name {
				return &(*c)[i]
			}

			if slices.Contains(objectDefinition.Interfaces, cf.ObjectDefinition.Name) {
				return &(*c)[i]
			}
			if slices.Contains(cf.ObjectDefinition.Interfaces, objectDefinition.Name) {
				return &(*c)[i]
			}
		}
	}

	f := creator()

	*c = append(*c, f)
	return &(*c)[len(*c)-1]
}

func shouldIncludeNode(directives ast.DirectiveList, variables map[string]any) bool {
	if len(directives) == 0 {
		return true
	}

	skip, include := false, true

	if d := directives.ForName("skip"); d != nil {
		skip = resolveIfArgument(d, variables)
	}

	if d := directives.ForName("include"); d != nil {
		include = resolveIfArgument(d, variables)
	}

	return !skip && include
}

func deferrable(
	directives ast.DirectiveList,
	variables map[string]any,
) (shouldDefer bool, label string) {
	d := directives.ForName("defer")
	if d == nil {
		return false, ""
	}

	shouldDefer = true

	for _, arg := range d.Arguments {
		switch arg.Name {
		case "if":
			if value, err := arg.Value.Value(variables); err == nil {
				shouldDefer, _ = value.(bool)
			}
		case "label":
			if value, err := arg.Value.Value(variables); err == nil {
				label, _ = value.(string)
			}
		default:
			panic(fmt.Sprintf("defer: argument '%s' not supported", arg.Name))
		}
	}

	return shouldDefer, label
}

func resolveIfArgument(d *ast.Directive, variables map[string]any) bool {
	arg := d.Arguments.ForName("if")
	if arg == nil {
		panic(fmt.Sprintf("%s: argument 'if' not defined", d.Name))
	}
	value, err := arg.Value.Value(variables)
	if err != nil {
		panic(err)
	}
	ret, ok := value.(bool)
	if !ok {
		panic(fmt.Sprintf("%s: argument 'if' is not a boolean", d.Name))
	}
	return ret
}
