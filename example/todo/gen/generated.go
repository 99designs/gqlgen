package gen

import (
	schema "github.com/vektah/graphql-go/schema"
	introspection "github.com/vektah/graphql-go/introspection"
	todo "github.com/vektah/graphql-go/example/todo"
	exec "github.com/vektah/graphql-go/exec"
	jsonw "github.com/vektah/graphql-go/jsonw"
	query "github.com/vektah/graphql-go/query"
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
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "createTodo":
		res, err := r.resolvers.Mutation_createTodo(
			arguments["text"].(string),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := ec.ExecuteSelectionSet(sels, r.Todo, &res)
		return json
	
	case "updateTodo":
		res, err := r.resolvers.Mutation_updateTodo(
			arguments["id"].(int),
			arguments["done"].(bool),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := ec.ExecuteSelectionSet(sels, r.Todo, &res)
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) Query(ec *exec.ExecutionContext, it interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "todo":
		res, err := r.resolvers.Query_todo(
			arguments["id"].(int),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := ec.ExecuteSelectionSet(sels, r.Todo, res)
		return json
	
	case "lastTodo":
		res, err := r.resolvers.Query_lastTodo()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := ec.ExecuteSelectionSet(sels, r.Todo, res)
		return json
	
	case "todos":
		res, err := r.resolvers.Query_todos()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := jsonw.Array{}
		for _, val := range res {
			json1 := ec.ExecuteSelectionSet(sels, r.Todo, &val)
			json = append(json, json1)
		}
		return json
	
	case "__schema":
		res := ec.IntrospectSchema()
		json := ec.ExecuteSelectionSet(sels, r.__Schema, res)
		return json
	
	case "__type":
		res := ec.IntrospectType(
			arguments["name"].(string),
		)
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) Todo(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*todo.Todo)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "id":
		res := jsonw.Int(it.ID)
		return res
	
	case "text":
		res := jsonw.String(it.Text)
		return res
	
	case "done":
		res := jsonw.Bool(it.Done)
		return res
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) __Directive(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.Directive)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "name":
		res := it.Name()
		json := jsonw.String(res)
		return json
	
	case "description":
		res := it.Description()
		json := jsonw.String(*res)
		return json
	
	case "locations":
		res := it.Locations()
		json := jsonw.Array{}
		for _, val := range res {
			json1 := jsonw.String(val)
			json = append(json, json1)
		}
		return json
	
	case "args":
		res := it.Args()
		json := jsonw.Array{}
		for _, val := range res {
			json1 := ec.ExecuteSelectionSet(sels, r.__InputValue, val)
			json = append(json, json1)
		}
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) __EnumValue(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.EnumValue)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "name":
		res := it.Name()
		json := jsonw.String(res)
		return json
	
	case "description":
		res := it.Description()
		json := jsonw.String(*res)
		return json
	
	case "isDeprecated":
		res := it.IsDeprecated()
		json := jsonw.Bool(res)
		return json
	
	case "deprecationReason":
		res := it.DeprecationReason()
		json := jsonw.String(*res)
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) __Field(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.Field)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "name":
		res := it.Name()
		json := jsonw.String(res)
		return json
	
	case "description":
		res := it.Description()
		json := jsonw.String(*res)
		return json
	
	case "args":
		res := it.Args()
		json := jsonw.Array{}
		for _, val := range res {
			json1 := ec.ExecuteSelectionSet(sels, r.__InputValue, val)
			json = append(json, json1)
		}
		return json
	
	case "type":
		res := it.Type()
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	case "isDeprecated":
		res := it.IsDeprecated()
		json := jsonw.Bool(res)
		return json
	
	case "deprecationReason":
		res := it.DeprecationReason()
		json := jsonw.String(*res)
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) __InputValue(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.InputValue)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "name":
		res := it.Name()
		json := jsonw.String(res)
		return json
	
	case "description":
		res := it.Description()
		json := jsonw.String(*res)
		return json
	
	case "type":
		res := it.Type()
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	case "defaultValue":
		res := it.DefaultValue()
		json := jsonw.String(*res)
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) __Schema(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.Schema)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "types":
		res := it.Types()
		json := jsonw.Array{}
		for _, val := range res {
			json1 := ec.ExecuteSelectionSet(sels, r.__Type, val)
			json = append(json, json1)
		}
		return json
	
	case "queryType":
		res := it.QueryType()
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	case "mutationType":
		res := it.MutationType()
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	case "subscriptionType":
		res := it.SubscriptionType()
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	case "directives":
		res := it.Directives()
		json := jsonw.Array{}
		for _, val := range res {
			json1 := ec.ExecuteSelectionSet(sels, r.__Directive, val)
			json = append(json, json1)
		}
		return json
	
	}
	panic("unknown field " + field)
}

func (r *resolvers) __Type(ec *exec.ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.Type)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "kind":
		res := it.Kind()
		json := jsonw.String(res)
		return json
	
	case "name":
		res := it.Name()
		json := jsonw.String(*res)
		return json
	
	case "description":
		res := it.Description()
		json := jsonw.String(*res)
		return json
	
	case "fields":
		res := it.Fields(
			arguments["includeDeprecated"].(bool),
		)
		json := jsonw.Array{}
		for _, val := range *res {
			json1 := ec.ExecuteSelectionSet(sels, r.__Field, val)
			json = append(json, json1)
		}
		return json
	
	case "interfaces":
		res := it.Interfaces()
		json := jsonw.Array{}
		for _, val := range *res {
			json1 := ec.ExecuteSelectionSet(sels, r.__Type, val)
			json = append(json, json1)
		}
		return json
	
	case "possibleTypes":
		res := it.PossibleTypes()
		json := jsonw.Array{}
		for _, val := range *res {
			json1 := ec.ExecuteSelectionSet(sels, r.__Type, val)
			json = append(json, json1)
		}
		return json
	
	case "enumValues":
		res := it.EnumValues(
			arguments["includeDeprecated"].(bool),
		)
		json := jsonw.Array{}
		for _, val := range *res {
			json1 := ec.ExecuteSelectionSet(sels, r.__EnumValue, val)
			json = append(json, json1)
		}
		return json
	
	case "inputFields":
		res := it.InputFields()
		json := jsonw.Array{}
		for _, val := range *res {
			json1 := ec.ExecuteSelectionSet(sels, r.__InputValue, val)
			json = append(json, json1)
		}
		return json
	
	case "ofType":
		res := it.OfType()
		json := ec.ExecuteSelectionSet(sels, r.__Type, res)
		return json
	
	}
	panic("unknown field " + field)
}

var Schema = schema.MustParse("\nschema {\n\tquery: Query\n\tmutation: Mutation\n}\n\ntype Query {\n\ttodo(id: Int!): Todo\n\tlastTodo: Todo\n\ttodos: [Todo!]!\n}\n\ntype Mutation {\n\tcreateTodo(text: String!): Todo!\n\tupdateTodo(id: Int!, done: Boolean!): Todo!\n}\n\ntype Todo {\n\tid: Int!\n\ttext: String!\n\tdone: Boolean!\n}\n")
