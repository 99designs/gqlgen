//go:generate gorunpkg github.com/99designs/gqlgen

package gqlapollotracingtest

import (
	"context"
	"fmt"
)

func NewResolver() *Resolver {
	return &Resolver{
		todos: []Todo{
			{
				ID:   "Todo:1",
				Text: "Play with cat",
				Done: true,
				User: User{
					ID:   "User:foobar",
					Name: "foobar",
				},
			},
		},
	}
}

type Resolver struct {
	todos []Todo
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateTodo(ctx context.Context, input NewTodo) (Todo, error) {
	todo := Todo{
		ID:   fmt.Sprintf("Todo:%d", len(r.todos)+1),
		Text: input.Text,
		User: User{
			ID:   input.UserID,
			Name: input.UserID,
		},
	}
	r.todos = append(r.todos, todo)
	return todo, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}
