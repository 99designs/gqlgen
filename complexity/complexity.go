package complexity

import (
	"context"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

func Calculate(ctx context.Context, es graphql.ExecutableSchema, op *ast.OperationDefinition, vars map[string]any) int {
	walker := complexityWalker{
		es:     es,
		schema: es.Schema(),
		vars:   vars,
	}
	return walker.selectionSetComplexity(ctx, op.SelectionSet)
}

type complexityWalker struct {
	es     graphql.ExecutableSchema
	schema *ast.Schema
	vars   map[string]any
}

func (cw complexityWalker) selectionSetComplexity(ctx context.Context, selectionSet ast.SelectionSet) int {
	var complexity int
	for _, selection := range selectionSet {
		switch s := selection.(type) {
		case *ast.Field:
			fieldDefinition := cw.schema.Types[s.Definition.Type.Name()]

			if fieldDefinition.Name == "__Schema" {
				continue
			}

			var childComplexity int
			switch fieldDefinition.Kind {
			case ast.Object, ast.Interface, ast.Union:
				childComplexity = cw.selectionSetComplexity(ctx, s.SelectionSet)
			}

			args := s.ArgumentMap(cw.vars)
			var fieldComplexity int
			if s.ObjectDefinition.Kind == ast.Interface {
				fieldComplexity = cw.interfaceFieldComplexity(ctx, s.ObjectDefinition, s.Name, childComplexity, args)
			} else {
				fieldComplexity = cw.fieldComplexity(ctx, s.ObjectDefinition.Name, s.Name, childComplexity, args)
			}
			complexity = safeAdd(complexity, fieldComplexity)

		case *ast.FragmentSpread:
			complexity = safeAdd(complexity, cw.selectionSetComplexity(ctx, s.Definition.SelectionSet))

		case *ast.InlineFragment:
			complexity = safeAdd(complexity, cw.selectionSetComplexity(ctx, s.SelectionSet))
		}
	}
	return complexity
}

func (cw complexityWalker) interfaceFieldComplexity(ctx context.Context, def *ast.Definition, field string, childComplexity int, args map[string]any) int {
	// Interfaces don't have their own separate field costs, so they have to assume the worst case.
	// We iterate over all implementors and choose the most expensive one.
	maxComplexity := 0
	implementors := cw.schema.GetPossibleTypes(def)
	for _, t := range implementors {
		fieldComplexity := cw.fieldComplexity(ctx, t.Name, field, childComplexity, args)
		if fieldComplexity > maxComplexity {
			maxComplexity = fieldComplexity
		}
	}
	return maxComplexity
}

func (cw complexityWalker) fieldComplexity(ctx context.Context, object, field string, childComplexity int, args map[string]any) int {
	if customComplexity, ok := cw.es.Complexity(ctx, object, field, childComplexity, args); ok && customComplexity >= childComplexity {
		return customComplexity
	}
	// default complexity calculation
	return safeAdd(1, childComplexity)
}

const maxInt = int(^uint(0) >> 1)

// safeAdd is a saturating add of a and b that ignores negative operands.
// If a + b would overflow through normal Go addition,
// it returns the maximum integer value instead.
//
// Adding complexities with this function prevents attackers from intentionally
// overflowing the complexity calculation to allow overly-complex queries.
//
// It also helps mitigate the impact of custom complexities that accidentally
// return negative values.
func safeAdd(a, b int) int {
	// Ignore negative operands.
	if a < 0 {
		if b < 0 {
			return 1
		}
		return b
	} else if b < 0 {
		return a
	}

	c := a + b
	if c < a {
		// Set c to maximum integer instead of overflowing.
		c = maxInt
	}
	return c
}
