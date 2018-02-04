//go:generate ggraphqlc -package gen -out gen/generated.go

package todo

import (
	"errors"
)

type Todo struct {
	ID     int
	Text   string
	Done   bool
	UserID int
}

type TodoResolver struct {
	todos  []Todo
	lastID int
}

func NewResolver() *TodoResolver {
	return &TodoResolver{
		todos: []Todo{
			{ID: 1, Text: "A todo not to forget", Done: false, UserID: 1},
			{ID: 2, Text: "This is the most important", Done: false, UserID: 1},
			{ID: 3, Text: "Please do this or else", Done: false, UserID: 1},
		},
		lastID: 3,
	}
}

func (r *TodoResolver) Query_todo(id int) (*Todo, error) {
	for _, todo := range r.todos {
		if todo.ID == id {
			return &todo, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *TodoResolver) Query_lastTodo() (*Todo, error) {
	if len(r.todos) == 0 {
		return nil, errors.New("not found")
	}
	return &r.todos[len(r.todos)-1], nil
}

func (r *TodoResolver) Query_todos() ([]Todo, error) {
	return r.todos, nil
}

func (r *TodoResolver) Mutation_createTodo(text string) (Todo, error) {
	newID := r.id()

	newTodo := Todo{
		ID:   newID,
		Text: text,
		Done: false,
	}

	r.todos = append(r.todos, newTodo)

	return newTodo, nil
}

func (r *TodoResolver) Mutation_updateTodo(id int, done bool) (Todo, error) {
	var affectedTodo *Todo

	for i := 0; i < len(r.todos); i++ {
		if r.todos[i].ID == id {
			r.todos[i].Done = done
			affectedTodo = &r.todos[i]
			break
		}
	}
	return *affectedTodo, errors.New("not found")
}

func (r *TodoResolver) id() int {
	r.lastID++
	return r.lastID
}
