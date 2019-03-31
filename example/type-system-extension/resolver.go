//go:generate go run ../../testdata/gqlgen.go

package type_system_extension

import (
	"context"
	"fmt"
)

func NewRootResolver() ResolverRoot {
	return &resolver{
		todos: []*Todo{
			{
				ID:       "Todo:1",
				Text:     "Buy a cat food",
				State:    StateNotYet,
				Verified: false,
			},
			{
				ID:       "Todo:2",
				Text:     "Check cat water",
				State:    StateDone,
				Verified: true,
			},
			{
				ID:       "Todo:3",
				Text:     "Check cat meal",
				State:    StateDone,
				Verified: true,
			},
		},
	}
}

type resolver struct {
	todos []*Todo
}

func (r *resolver) MyQuery() MyQueryResolver {
	return &queryResolver{r}
}

func (r *resolver) MyMutation() MyMutationResolver {
	return &mutationResolver{r}
}

type queryResolver struct{ *resolver }

func (r *queryResolver) Todos(ctx context.Context) ([]Todo, error) {
	todos := make([]Todo, 0, len(r.todos))
	for _, todo := range r.todos {
		todos = append(todos, *todo)
	}
	return todos, nil
}

func (r *queryResolver) Todo(ctx context.Context, id string) (*Todo, error) {
	for _, todo := range r.todos {
		if todo.ID == id {
			return todo, nil
		}
	}
	return nil, nil
}

type mutationResolver struct{ *resolver }

func (r *mutationResolver) CreateTodo(ctx context.Context, todoInput TodoInput) (*Todo, error) {
	newID := fmt.Sprintf("Todo:%d", len(r.todos)+1)
	newTodo := &Todo{
		ID:    newID,
		Text:  todoInput.Text,
		State: StateNotYet,
	}
	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}
