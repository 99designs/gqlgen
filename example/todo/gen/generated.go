package gen

import (
	exec "github.com/vektah/graphql-go/exec"
	jsonw "github.com/vektah/graphql-go/jsonw"
	query "github.com/vektah/graphql-go/query"
	schema "github.com/vektah/graphql-go/schema"
	todo "github.com/vektah/graphql-go/example/todo"
)

type Resolvers interface {
	Mutation_createTodo(text string) (todo.Todo, error)
	Mutation_updateTodo(id int,done bool) (todo.Todo, error)
	Query_todo(id int) (*todo.Todo, error)
	Query_lastTodo() (*todo.Todo, error)
	Query_todos() ([]todo.Todo, error)
}

func NewResolver(r Resolvers) exec.Root {
	return &resolvers{r}
}

type resolvers struct {
	resolvers Resolvers
}

func (r *resolvers) Mutation(ec *exec.ExecutionContext, it interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	switch field {
	case "createTodo":
		result, err := r.resolvers.Mutation_createTodo(
			arguments["text"].(string),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.Todo, &result)
	
	case "updateTodo":
		result, err := r.resolvers.Mutation_updateTodo(
			arguments["id"].(int),
			arguments["done"].(bool),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.Todo, &result)
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) Query(ec *exec.ExecutionContext, it interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	switch field {
	case "todo":
		result, err := r.resolvers.Query_todo(
			arguments["id"].(int),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.Todo, result)
	
	case "lastTodo":
		result, err := r.resolvers.Query_lastTodo()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.Todo, result)
	
	case "todos":
		result, err := r.resolvers.Query_todos()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		return ec.ExecuteSelectionSet(sels, r.Todo, &result)
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) Todo(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*todo.Todo)
	switch field {
	case "id":
		return jsonw.ID(it.ID)
	
	case "text":
		return jsonw.String(it.Text)
	
	case "done":
		return jsonw.Boolean(it.Done)
	
	}
	panic("unknown field " + field)
}

var Schema = schema.MustParse("\nschema {\n\tquery: Query\n\tmutation: Mutation\n}\n\ntype Query {\n\ttodo(id: Int!): Todo\n\tlastTodo: Todo\n\ttodos: [Todo!]!\n}\n\ntype Mutation {\n\tcreateTodo(text: String!): Todo!\n\tupdateTodo(id: Int!, done: Boolean!): Todo!\n}\n\ntype Todo {\n\tid: ID!\n\ttext: String!\n\tdone: Boolean!\n}\n")
