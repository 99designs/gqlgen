//go:generate ggraphqlc -out generated.go

package todo

import (
	"context"
	"errors"
	"time"

	"github.com/mitchellh/mapstructure"
)

type Todo struct {
	ID     int
	Text   string
	Done   bool
	UserID int
}

type todoResolver struct {
	todos  []Todo
	lastID int
}

func New() *todoResolver {
	return &todoResolver{
		todos: []Todo{
			{ID: 1, Text: "A todo not to forget", Done: false, UserID: 1},
			{ID: 2, Text: "This is the most important", Done: false, UserID: 1},
			{ID: 3, Text: "Please do this or else", Done: false, UserID: 1},
		},
		lastID: 3,
	}
}

func (r *todoResolver) MyQuery_todo(ctx context.Context, id int) (*Todo, error) {
	time.Sleep(220 * time.Millisecond)
	for _, todo := range r.todos {
		if todo.ID == id {
			return &todo, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *todoResolver) MyQuery_lastTodo(ctx context.Context) (*Todo, error) {
	if len(r.todos) == 0 {
		return nil, errors.New("not found")
	}
	return &r.todos[len(r.todos)-1], nil
}

func (r *todoResolver) MyQuery_todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}

func (r *todoResolver) MyMutation_createTodo(ctx context.Context, text string) (Todo, error) {
	newID := r.id()

	newTodo := Todo{
		ID:   newID,
		Text: text,
		Done: false,
	}

	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}

// this example uses a map instead of a struct for the change set. this scales updating keys on large objects where
// most properties are optional, and if unspecified the existing value should be kept.
func (r *todoResolver) MyMutation_updateTodo(ctx context.Context, id int, changes map[string]interface{}) (*Todo, error) {
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

func (r *todoResolver) id() int {
	r.lastID++
	return r.lastID
}
