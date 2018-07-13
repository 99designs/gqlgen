//go:generate gorunpkg github.com/vektah/gqlgen

package scalars

import (
	context "context"
	"external"
	"fmt"
	time "time"

	"github.com/vektah/gqlgen/example/scalars/model"
)

type Resolver struct {
}

func (r *Resolver) Query_user(ctx context.Context, id external.ObjectID) (*model.User, error) {
	return &model.User{
		ID:      id,
		Name:    fmt.Sprintf("Test User %d", id),
		Created: time.Now(),
		Address: model.Address{ID: 1, Location: &model.Point{1, 2}},
		Tier:    model.TierC,
	}, nil
}

func (r *Resolver) Query_search(ctx context.Context, input model.SearchArgs) ([]model.User, error) {
	location := model.Point{1, 2}
	if input.Location != nil {
		location = *input.Location
	}

	created := time.Now()
	if input.CreatedAfter != nil {
		created = *input.CreatedAfter
	}

	return []model.User{
		{
			ID:      1,
			Name:    "Test User 1",
			Created: created,
			Address: model.Address{ID: 2, Location: &location},
			Tier:    model.TierA,
		},
		{
			ID:      2,
			Name:    "Test User 2",
			Created: created,
			Address: model.Address{ID: 1, Location: &location},
			Tier:    model.TierC,
		},
	}, nil
}

func (r *Resolver) User_primitiveResolver(ctx context.Context, obj *model.User) (string, error) {
	return "test", nil
}

func (r *Resolver) User_customResolver(ctx context.Context, obj *model.User) (model.Point, error) {
	return model.Point{5, 1}, nil
}
