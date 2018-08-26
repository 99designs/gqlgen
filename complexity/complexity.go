package complexity

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"
)

func Calculate(es graphql.ExecutableSchema, op *ast.OperationDefinition, vars map[string]interface{}) int {
	walker := complexityWalker{
		es:   es,
		vars: vars,
	}
	return walker.selectionSetComplexity(op.SelectionSet)
}

type complexityWalker struct {
	es   graphql.ExecutableSchema
	vars map[string]interface{}
}

func (cw complexityWalker) selectionSetComplexity(selectionSet ast.SelectionSet) int {
	var complexity int
	for _, selection := range selectionSet {
		switch s := selection.(type) {
		case *ast.Field:
			var childComplexity int
			switch s.ObjectDefinition.Kind {
			case ast.Object, ast.Interface, ast.Union:
				childComplexity = cw.selectionSetComplexity(s.SelectionSet)
			}

			args := s.ArgumentMap(cw.vars)
			if customComplexity, ok := cw.es.Complexity(s.ObjectDefinition.Name, s.Name, childComplexity, args); ok && customComplexity >= childComplexity {
				complexity = safeAdd(complexity, customComplexity)
			} else {
				// default complexity calculation
				complexity = safeAdd(complexity, safeAdd(1, childComplexity))
			}

		case *ast.FragmentSpread:
			complexity = safeAdd(complexity, cw.selectionSetComplexity(s.Definition.SelectionSet))

		case *ast.InlineFragment:
			complexity = safeAdd(complexity, cw.selectionSetComplexity(s.SelectionSet))
		}
	}
	return complexity
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
