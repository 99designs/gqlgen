//go:generate gorunpkg github.com/vektah/gqlgen -typemap types.json -out generated.go

package scalars

import (
	context "context"
	"time"
)

type Resolver struct {
}

func (r *Resolver) Query_user(ctx context.Context, id string) (*User, error) {
	return &User{
		ID:       id,
		Name:     "Test User " + id,
		Created:  time.Now(),
		Location: Point{1, 2},
	}, nil
}

func (r *Resolver) Query_search(ctx context.Context, input SearchArgs) ([]User, error) {
	location := Point{1, 2}
	if input.Location != nil {
		location = *input.Location
	}

	created := time.Now()
	if input.CreatedAfter != nil {
		created = *input.CreatedAfter
	}

	return []User{
		{
			ID:       "1",
			Name:     "Test User 1",
			Created:  created,
			Location: location,
		},
		{
			ID:       "2",
			Name:     "Test User 2",
			Created:  created,
			Location: location,
		},
	}, nil
}
