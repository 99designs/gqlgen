//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go

package selection

import (
	context "context"
	"fmt"
	"time"

	"github.com/vektah/gqlgen/graphql"
	query "github.com/vektah/gqlgen/neelance/query"
)

type SelectionResolver struct{}

func (r *SelectionResolver) Query_events(ctx context.Context) ([]Event, error) {
	var sels []string

	reqCtx := graphql.GetRequestContext(ctx)
	fieldSelections := graphql.GetResolverContext(ctx).Field.Selections
	for _, sel := range fieldSelections {
		switch sel := sel.(type) {
		case *query.Field:
			sels = append(sels, fmt.Sprintf("%s as %s", sel.Name.Name, sel.Alias.Name))
		case *query.InlineFragment:
			sels = append(sels, fmt.Sprintf("inline fragment on %s", sel.On.Name))
		case *query.FragmentSpread:
			fragment := reqCtx.Doc.Fragments.Get(sel.Name.Name)
			sels = append(sels, fmt.Sprintf("named fragment %s on %s", sel.Name.Name, fragment.On.Name))
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
