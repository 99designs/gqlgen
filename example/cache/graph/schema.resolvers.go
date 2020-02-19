// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"errors"
	"time"

	"github.com/99designs/gqlgen/example/cache/graph/generated"
	"github.com/99designs/gqlgen/example/cache/graph/model"
	"github.com/99designs/gqlgen/graphql"
)

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	graphql.SetCacheHint(ctx, graphql.CacheScopePublic, 200*time.Second)
	return []*model.Todo{
		{"1", "Todo1", false, nil},
		{"2", "Todo2", true, nil},
		{"3", "Todo3", false, nil},
	}, nil
}

func (r *queryResolver) Todo(ctx context.Context, id string) (*model.Todo, error) {
	graphql.SetCacheHint(ctx, graphql.CacheScopePublic, 100*time.Second)
	switch id {
	case "1":
		return &model.Todo{ID: "1", Text: "Todo1"}, nil
	case "2":
		return &model.Todo{ID: "2", Text: "Todo2"}, nil
	default:
		return nil, errors.New("not found")
	}
}

func (r *todoResolver) User(ctx context.Context, obj *model.Todo) (*model.User, error) {
	if obj.ID == "1" {
		graphql.SetCacheHint(ctx, graphql.CacheScopePublic, 50*time.Second)
		return &model.User{
			ID:   "1",
			Name: "User 1",
		}, nil
	}

	if obj.ID == "2" {
		graphql.SetCacheHint(ctx, graphql.CacheScopePrivate, 20*time.Second)
		return &model.User{
			ID:   "2",
			Name: "User 2",
		}, nil
	}
	return nil, errors.New("ERR")
}

func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }
func (r *Resolver) Todo() generated.TodoResolver   { return &todoResolver{r} }

type queryResolver struct{ *Resolver }
type todoResolver struct{ *Resolver }
