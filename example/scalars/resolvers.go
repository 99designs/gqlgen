//go:generate gorunpkg github.com/vektah/gqlgen -typemap types.json -out generated.go

package scalars

import (
	context "context"
	"fmt"
	"time"

	"external"
)

type Resolver struct {
}

func (r *Resolver) Query_user(ctx context.Context, id external.ObjectID) (*User, error) {
	return &User{
		ID:      id,
		Name:    fmt.Sprintf("Test User %d", id),
		Created: time.Now(),
		Address: Address{ID: 1, Location: &Point{1, 2}},
		Tier:    TierC,
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
			ID:      1,
			Name:    "Test User 1",
			Created: created,
			Address: Address{ID: 2, Location: &location},
			Tier:    TierA,
		},
		{
			ID:      2,
			Name:    "Test User 2",
			Created: created,
			Address: Address{ID: 1, Location: &location},
			Tier:    TierC,
		},
	}, nil
}

func (r *Resolver) User_primitiveResolver(ctx context.Context, obj *User) (string, error) {
	return "test", nil
}

func (r *Resolver) User_customResolver(ctx context.Context, obj *User) (Point, error) {
	return Point{5, 1}, nil
}
