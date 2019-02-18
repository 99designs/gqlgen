//go:generate go run ../../testdata/gqlgen.go

package scalars

import (
	context "context"
	"fmt"
	time "time"

	"github.com/99designs/gqlgen/example/scalars/external"
	"github.com/99designs/gqlgen/example/scalars/model"
)

type Resolver struct {
}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

func (r *Resolver) User() UserResolver {
	return &userResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) User(ctx context.Context, id external.ObjectID) (*model.User, error) {
	return &model.User{
		ID:      id,
		Name:    fmt.Sprintf("Test User %d", id),
		Created: time.Now(),
		Address: model.Address{ID: 1, Location: &model.Point{X: 1, Y: 2}},
		Tier:    model.TierC,
	}, nil
}

func (r *queryResolver) Search(ctx context.Context, input *model.SearchArgs) ([]model.User, error) {
	location := model.Point{X: 1, Y: 2}
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

type userResolver struct{ *Resolver }

func (r *userResolver) PrimitiveResolver(ctx context.Context, obj *model.User) (string, error) {
	return "test", nil
}

func (r *userResolver) CustomResolver(ctx context.Context, obj *model.User) (*model.Point, error) {
	return &model.Point{X: 5, Y: 1}, nil
}
