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
	typeName := ""
	switch op.Operation {
	case ast.Query:
		typeName = es.Schema().Query.Name
	case ast.Mutation:
		typeName = es.Schema().Mutation.Name
	case ast.Subscription:
		typeName = es.Schema().Subscription.Name
	}
	return walker.selectionSetComplexity(typeName, op.SelectionSet)
}

type complexityWalker struct {
	es   graphql.ExecutableSchema
	vars map[string]interface{}
}

func (cw complexityWalker) selectionSetComplexity(typeName string, selectionSet ast.SelectionSet) int {
	var complexity int
	for _, selection := range selectionSet {
		switch s := selection.(type) {
		case *ast.Field:
			var childComplexity int
			switch s.ObjectDefinition.Kind {
			case ast.Object, ast.Interface, ast.Union:
				childComplexity = cw.selectionSetComplexity(s.ObjectDefinition.Name, s.SelectionSet)
			}

			args := s.ArgumentMap(cw.vars)
			if customComplexity, ok := cw.es.Complexity(typeName, s.Name, childComplexity, args); ok {
				complexity = safeAdd(complexity, customComplexity)
			} else {
				// default complexity calculation
				complexity = safeAdd(complexity, safeAdd(1, childComplexity))
			}

		case *ast.FragmentSpread:
			complexity = safeAdd(complexity, cw.selectionSetComplexity(typeName, s.Definition.SelectionSet))

		case *ast.InlineFragment:
			complexity = safeAdd(complexity, cw.selectionSetComplexity(typeName, s.SelectionSet))
		}
	}
	return complexity
}

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
	if a <= 0 {
		if b < 0 {
			return 0
		}
		return b
	} else if b <= 0 {
		return a
	}

	c := a + b
	if c < a {
		// Set c to maximum integer instead of overflowing.
		c = int(^uint(0) >> 1)
	}
	return c
}
