// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package config

import (
	"context"
)

func (r *mutationResolver) CreateTodo(ctx context.Context, input NewTodo) (*Todo, error) {
	newID := r.nextID
	r.nextID++

	newTodo := &Todo{
		DatabaseID:  newID,
		Description: input.Text,
	}

	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}
func (r *queryResolver) Todos(ctx context.Context) ([]*Todo, error) {
	return r.todos, nil
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
