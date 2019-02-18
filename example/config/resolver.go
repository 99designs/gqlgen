//go:generate go run ../../testdata/gqlgen.go

package config

import (
	"context"
	"fmt"
)

func New() Config {
	c := Config{
		Resolvers: &Resolver{
			todos: []Todo{
				{DatabaseID: 1, Description: "A todo not to forget", Done: false},
				{DatabaseID: 2, Description: "This is the most important", Done: false},
				{DatabaseID: 3, Description: "Please do this or else", Done: false},
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
func (r *Resolver) Todo() TodoResolver {
	return &todoResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateTodo(ctx context.Context, input NewTodo) (*Todo, error) {
	newID := r.nextID
	r.nextID++

	newTodo := Todo{
		DatabaseID:  newID,
		Description: input.Text,
	}

	r.todos = append(r.todos, newTodo)

	return &newTodo, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}

type todoResolver struct{ *Resolver }

func (r *todoResolver) Description(ctx context.Context, obj *Todo) (string, error) {
	panic("implement me")
}

func (r *todoResolver) ID(ctx context.Context, obj *Todo) (string, error) {
	if obj.ID != "" {
		return obj.ID, nil
	}

	obj.ID = fmt.Sprintf("TODO:%d", obj.DatabaseID)

	return obj.ID, nil
}
