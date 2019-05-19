package structvalslices

import (
	"context"
	"errors"
) // THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

// Resolver is a test resolver
type Resolver struct{}

// Query resolves the schema's queries.
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Pages(ctx context.Context, tags []TagInput) ([]Page, error) {
	if len(tags) == 0 {
		return nil, errors.New("No tags")
	}

	tags2 := []Tag{
		{
			Kind: tags[0].Kind,
			Name: tags[0].Name,
		},
	}

	return []Page{
		{
			ID:   "P00000007",
			Str:  "Stuff and things",
			Tags: tags2,
		},
	}, nil
}
