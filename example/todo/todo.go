//go:generate gorunpkg github.com/vektah/gqlgen --out generated.go -v

package todo

import (
	"context"
	"errors"
	"time"

	"github.com/mitchellh/mapstructure"
)

func New() Config {
	r := &resolvers{
		todos: []Todo{
			{ID: 1, Text: "A todo not to forget", Done: false},
			{ID: 2, Text: "This is the most important", Done: false},
			{ID: 3, Text: "Please do this or else", Done: false},
		},
		lastID: 3,
	}
	return Config{Resolvers: r}
}

type resolvers struct {
	todos  []Todo
	lastID int
}

func (r *resolvers) MyQuery() MyQueryResolver {
	return (*QueryResolver)(r)
}

func (r *resolvers) MyMutation() MyMutationResolver {
	return (*MutationResolver)(r)
}

type QueryResolver resolvers

func (r *QueryResolver) Todo(ctx context.Context, id int) (*Todo, error) {
	time.Sleep(220 * time.Millisecond)

	if id == 666 {
		panic("critical failure")
	}

	for _, todo := range r.todos {
		if todo.ID == id {
			return &todo, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *QueryResolver) LastTodo(ctx context.Context) (*Todo, error) {
	if len(r.todos) == 0 {
		return nil, errors.New("not found")
	}
	return &r.todos[len(r.todos)-1], nil
}

func (r *QueryResolver) Todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}

type MutationResolver resolvers

func (r *MutationResolver) CreateTodo(ctx context.Context, todo TodoInput) (Todo, error) {
	newID := r.id()

	newTodo := Todo{
		ID:   newID,
		Text: todo.Text,
	}

	if todo.Done != nil {
		newTodo.Done = *todo.Done
	}

	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}

func (r *MutationResolver) UpdateTodo(ctx context.Context, id int, changes map[string]interface{}) (*Todo, error) {
	var affectedTodo *Todo

	for i := 0; i < len(r.todos); i++ {
		if r.todos[i].ID == id {
			affectedTodo = &r.todos[i]
			break
		}
	}

	if affectedTodo == nil {
		return nil, nil
	}

	err := mapstructure.Decode(changes, affectedTodo)
	if err != nil {
		panic(err)
	}

	return affectedTodo, nil
}

func (r *MutationResolver) id() int {
	r.lastID++
	return r.lastID
}
