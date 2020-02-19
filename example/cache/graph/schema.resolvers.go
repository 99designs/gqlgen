// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"

	"github.com/99designs/gqlgen/example/cache/graph/generated"
	"github.com/99designs/gqlgen/example/cache/graph/model"
)

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	return []*model.Todo{
		{"1", "Todo1", false},
		{"2", "Todo2", true},
		{"3", "Todo3", false},
	}, nil
}

func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
