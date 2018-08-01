//go:generate gorunpkg github.com/vektah/gqlgen

package config

import (
	"context"
	"fmt"
)

func New() Config {
	c := Config{
		Resolvers: &Resolver{
			todos: []Todo{
				{ID: "TODO:1", Description: "A todo not to forget", Done: false},
				{ID: "TODO:2", Description: "This is the most important", Done: false},
				{ID: "TODO:3", Description: "Please do this or else", Done: false},
			},
			nextID: 3,
		},
	}
	return c
}

type Resolver struct {
	todos  []Todo
	nextID int
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateTodo(ctx context.Context, input NewTodo) (Todo, error) {
	newID := r.nextID
	r.nextID++

	newTodo := Todo{
		ID:          fmt.Sprintf("TODO:%d", newID),
		Description: input.Text,
	}

	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}
