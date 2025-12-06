//go:generate go run ../../testdata/gqlgen.go

package unionextension

import (
	"context"
)

type Resolver struct{}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Events(ctx context.Context) ([]Event, error) {
	return []Event{&Like{From: "John"}, &Post{Message: "Hello"}}, nil
}

func (r *queryResolver) CachedEvents(ctx context.Context) ([]Event, error) {
	return []Event{&CachedLike{}, &CachedPost{}}, nil
}
