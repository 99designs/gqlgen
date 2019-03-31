//go:generate go run ../../testdata/gqlgen.go

package selection

import (
	context "context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"
)

type Resolver struct{}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Events(ctx context.Context) ([]Event, error) {
	var sels []string

	reqCtx := graphql.GetRequestContext(ctx)
	fieldSelections := graphql.GetResolverContext(ctx).Field.Selections
	for _, sel := range fieldSelections {
		switch sel := sel.(type) {
		case *ast.Field:
			sels = append(sels, fmt.Sprintf("%s as %s", sel.Name, sel.Alias))
		case *ast.InlineFragment:
			sels = append(sels, fmt.Sprintf("inline fragment on %s", sel.TypeCondition))
		case *ast.FragmentSpread:
			fragment := reqCtx.Doc.Fragments.ForName(sel.Name)
			sels = append(sels, fmt.Sprintf("named fragment %s on %s", sel.Name, fragment.TypeCondition))
		}
	}

	var events []Event
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			events = append(events, &Like{
				Selection: sels,
				Collected: formatCollected(graphql.CollectFieldsCtx(ctx, []string{"Like"})),
				Reaction:  ":=)",
				Sent:      time.Now(),
			})
		} else {
			events = append(events, &Post{
				Selection: sels,
				Collected: formatCollected(graphql.CollectFieldsCtx(ctx, []string{"Post"})),
				Message:   "Hey",
				Sent:      time.Now(),
			})
		}
	}

	return events, nil
}

func formatCollected(cf []graphql.CollectedField) []string {
	var res []string

	for _, f := range cf {
		res = append(res, fmt.Sprintf("%s as %s", f.Name, f.Alias))
	}
	return res
}
