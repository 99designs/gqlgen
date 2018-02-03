package gen

import (
	"fmt"

	"github.com/vektah/graphql-go/example/todo"
	"github.com/vektah/graphql-go/exec"
	"github.com/vektah/graphql-go/jsonw"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/schema"
)

type Resolvers interface {
	Query_todo(id int) (*todo.Todo, error)
	Query_lastTodo() (*todo.Todo, error)
	Query_todos() ([]*todo.Todo, error)

	Mutation_createTodo(text string) (todo.Todo, error)
	Mutation_updateTodo(id int, done bool) (*todo.Todo, error)
}

func NewResolver(r Resolvers) exec.Root {
	return &resolvers{r}
}

type resolvers struct {
	resolvers Resolvers
}

func (r *resolvers) Query(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	switch field {
	case "todo":
		result, err := r.resolvers.Query_todo(arguments["id"].(int))
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.todo, result)

	case "lastTodo":
		result, err := r.resolvers.Query_lastTodo()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.todo, result)

	case "todos":
		result, err := r.resolvers.Query_todos()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}

		var enc jsonw.Array
		for _, val := range result {
			enc = append(enc, ec.ExecuteSelectionSet(sels, r.todo, val))
		}

		return enc
	}

	panic("unknown field " + field)
}

func (r *resolvers) Mutation(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	switch field {
	case "createTodo":
		result, err := r.resolvers.Mutation_createTodo(arguments["text"].(string))
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.todo, result)

	case "updateTodo":
		result, err := r.resolvers.Mutation_updateTodo(arguments["id"].(int), arguments["done"].(bool))
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.todo, result)
	}

	panic("unknown field " + field)
}

func (r *resolvers) todo(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	fmt.Print("todoExec", object)
	switch field {
	case "id":
		return jsonw.Int(object.(*todo.Todo).ID)
	case "text":
		return jsonw.String(object.(*todo.Todo).Text)
	case "done":
		return jsonw.Bool(object.(*todo.Todo).Done)
	}
	return jsonw.Null
}

var Schema *schema.Schema

const schemaStr = `schema {
	query: Query
	mutation: Mutation
}
type Query {
	todo(id: Integer!): Todo
	lastTodo: Todo
	todos: [Todos!]!
	user(id: Integer!): User
}
type Mutation {
	createTodo(text: String!): Todo!
	updateTodo(id: Integer!, text: String!): Todo!
}
type Todo @go(type:"github.com/99designs/graphql-go/example/todo.Todo") {
	id: ID!
	text: String!
	done: Boolean!
	user: User!
}
type User @go(type:"github.com/99designs/graphql-go/example/todo.User"){
	id: ID!
	name: String!
}
`

func init() {
	Schema = schema.MustParse(schemaStr)
}
