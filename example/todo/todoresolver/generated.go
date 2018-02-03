package todoresolver

import (
	"fmt"

	"github.com/vektah/graphql-go/example/todo"
	"github.com/vektah/graphql-go/internal/query"
	"github.com/vektah/graphql-go/internal/schema"
	"github.com/vektah/graphql-go/jsonw"
)

type Resolvers interface {
	Query_todo(id int) (*todo.Todo, error)
	Query_lastTodo() (*todo.Todo, error)
	Query_todos() ([]*todo.Todo, error)

	Mutation_createTodo(text string) (todo.Todo, error)
	Mutation_updateTodo(id int, done bool) (*todo.Todo, error)
}

type (
	queryType    struct{}
	mutationType struct{}
	todoType     struct{}
)

func (q queryType) GetField(field string) Type {
	switch field {
	case "todo":
		return todoType{}
	case "lastTodo":
		return todoType{}
	case "todos":
		return todoType{}
	}
	return nil
}

func (q queryType) Execute(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	fmt.Println("query::exec")
	switch field {
	case "todo":
		result, err := ec.resolvers.Query_todo(arguments["id"].(int))
		if err != nil {
			ec.error(err)
			return jsonw.Null
		}
		return ec.executeSelectionSet(sels, todoType{}, result)

	case "lastTodo":
		result, err := ec.resolvers.Query_lastTodo()
		if err != nil {
			ec.error(err)
			return jsonw.Null
		}
		return ec.executeSelectionSet(sels, todoType{}, result)

	case "todos":
		result, err := ec.resolvers.Query_todos()
		if err != nil {
			ec.error(err)
			return jsonw.Null
		}

		var enc jsonw.Array
		for _, val := range result {
			enc = append(enc, ec.executeSelectionSet(sels, todoType{}, val))
		}

		return enc
	}

	panic("unknown field " + field)
}

func (q mutationType) GetField(field string) Type {
	switch field {
	case "createTodo":
		return todoType{}
	case "updateTodo":
		return todoType{}
	}
	return nil
}

func (q mutationType) Execute(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	switch field {
	case "createTodo":
		result, err := ec.resolvers.Mutation_createTodo(arguments["text"].(string))
		if err != nil {
			ec.error(err)
			return jsonw.Null
		}
		return ec.executeSelectionSet(sels, todoType{}, result)

	case "updateTodo":
		result, err := ec.resolvers.Mutation_updateTodo(arguments["id"].(int), arguments["done"].(bool))
		if err != nil {
			ec.error(err)
			return jsonw.Null
		}
		return ec.executeSelectionSet(sels, todoType{}, result)
	}

	panic("unknown field " + field)
}

func (q todoType) GetField(field string) Type {
	return nil
}

func (q todoType) Execute(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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

var parsedSchema *schema.Schema

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
	parsedSchema = schema.New()
	parsedSchema.Resolve(schemaStr)
}
