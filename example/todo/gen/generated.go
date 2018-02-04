package gen

import (
	"github.com/vektah/graphql-go/schema"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/example/todo"
	"github.com/vektah/graphql-go/jsonw"
	"github.com/vektah/graphql-go/query"
)

type Resolvers interface {
	Mutation_createTodo(text string) (todo.Todo, error)
	Mutation_updateTodo(id int,done bool) (todo.Todo, error)
	Query_todo(id int) (*todo.Todo, error)
	Query_lastTodo() (*todo.Todo, error)
	Query_todos() ([]todo.Todo, error)
}

type mutationType struct {}

func (mutationType) accepts(name string) bool {
	return true
}

func (mutationType) resolve(ec *executionContext, it interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "createTodo":
		res, err := ec.resolvers.Mutation_createTodo(
			arguments["text"].(string),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := ec.executeSelectionSet(sels, todoType{}, &res)
		return json
	
	case "updateTodo":
		res, err := ec.resolvers.Mutation_updateTodo(
			arguments["id"].(int),
			arguments["done"].(bool),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := ec.executeSelectionSet(sels, todoType{}, &res)
		return json
	
	}
	panic("unknown field " + field)
}

type queryType struct {}

func (queryType) accepts(name string) bool {
	return true
}

func (queryType) resolve(ec *executionContext, it interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "todo":
		res, err := ec.resolvers.Query_todo(
			arguments["id"].(int),
		)
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, todoType{}, res)
			json = json1
		}
		return json
	
	case "lastTodo":
		res, err := ec.resolvers.Query_lastTodo()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, todoType{}, res)
			json = json1
		}
		return json
	
	case "todos":
		res, err := ec.resolvers.Query_todos()
		if err != nil {
			ec.Error(err)
			return jsonw.Null
		}
		json := jsonw.Array{}
		for _, val := range res {
			json1 := ec.executeSelectionSet(sels, todoType{}, &val)
			json = append(json, json1)
		}
		return json
	
	case "__schema":
		res := ec.introspectSchema()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __SchemaType{}, res)
			json = json1
		}
		return json
	
	case "__type":
		res := ec.introspectType(
			arguments["name"].(string),
		)
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	}
	panic("unknown field " + field)
}

type todoType struct {}

func (todoType) accepts(name string) bool {
	return true
}

func (todoType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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

type __DirectiveType struct {}

func (__DirectiveType) accepts(name string) bool {
	return true
}

func (__DirectiveType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
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
			var json1 jsonw.Encodable = jsonw.Null
			if val != nil {
				json11 := ec.executeSelectionSet(sels, __InputValueType{}, val)
				json1 = json11
			}
			json = append(json, json1)
		}
		return json
	
	}
	panic("unknown field " + field)
}

type __EnumValueType struct {}

func (__EnumValueType) accepts(name string) bool {
	return true
}

func (__EnumValueType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	case "isDeprecated":
		res := it.IsDeprecated()
		json := jsonw.Bool(res)
		return json
	
	case "deprecationReason":
		res := it.DeprecationReason()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	}
	panic("unknown field " + field)
}

type __FieldType struct {}

func (__FieldType) accepts(name string) bool {
	return true
}

func (__FieldType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	case "args":
		res := it.Args()
		json := jsonw.Array{}
		for _, val := range res {
			var json1 jsonw.Encodable = jsonw.Null
			if val != nil {
				json11 := ec.executeSelectionSet(sels, __InputValueType{}, val)
				json1 = json11
			}
			json = append(json, json1)
		}
		return json
	
	case "type":
		res := it.Type()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	case "isDeprecated":
		res := it.IsDeprecated()
		json := jsonw.Bool(res)
		return json
	
	case "deprecationReason":
		res := it.DeprecationReason()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	}
	panic("unknown field " + field)
}

type __InputValueType struct {}

func (__InputValueType) accepts(name string) bool {
	return true
}

func (__InputValueType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	case "type":
		res := it.Type()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	case "defaultValue":
		res := it.DefaultValue()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	}
	panic("unknown field " + field)
}

type __SchemaType struct {}

func (__SchemaType) accepts(name string) bool {
	return true
}

func (__SchemaType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
	it := object.(*introspection.Schema)
	if it == nil {
		return jsonw.Null
	}
	switch field {
	case "types":
		res := it.Types()
		json := jsonw.Array{}
		for _, val := range res {
			var json1 jsonw.Encodable = jsonw.Null
			if val != nil {
				json11 := ec.executeSelectionSet(sels, __TypeType{}, val)
				json1 = json11
			}
			json = append(json, json1)
		}
		return json
	
	case "queryType":
		res := it.QueryType()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	case "mutationType":
		res := it.MutationType()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	case "subscriptionType":
		res := it.SubscriptionType()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	case "directives":
		res := it.Directives()
		json := jsonw.Array{}
		for _, val := range res {
			var json1 jsonw.Encodable = jsonw.Null
			if val != nil {
				json11 := ec.executeSelectionSet(sels, __DirectiveType{}, val)
				json1 = json11
			}
			json = append(json, json1)
		}
		return json
	
	}
	panic("unknown field " + field)
}

type __TypeType struct {}

func (__TypeType) accepts(name string) bool {
	return true
}

func (__TypeType) resolve(ec *executionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {
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
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	case "description":
		res := it.Description()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.String(*res)
			json = json1
		}
		return json
	
	case "fields":
		res := it.Fields(
			arguments["includeDeprecated"].(bool),
		)
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.Array{}
			for _, val := range *res {
				var json11 jsonw.Encodable = jsonw.Null
				if val != nil {
					json111 := ec.executeSelectionSet(sels, __FieldType{}, val)
					json11 = json111
				}
				json1 = append(json1, json11)
			}
			json = json1
		}
		return json
	
	case "interfaces":
		res := it.Interfaces()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.Array{}
			for _, val := range *res {
				var json11 jsonw.Encodable = jsonw.Null
				if val != nil {
					json111 := ec.executeSelectionSet(sels, __TypeType{}, val)
					json11 = json111
				}
				json1 = append(json1, json11)
			}
			json = json1
		}
		return json
	
	case "possibleTypes":
		res := it.PossibleTypes()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.Array{}
			for _, val := range *res {
				var json11 jsonw.Encodable = jsonw.Null
				if val != nil {
					json111 := ec.executeSelectionSet(sels, __TypeType{}, val)
					json11 = json111
				}
				json1 = append(json1, json11)
			}
			json = json1
		}
		return json
	
	case "enumValues":
		res := it.EnumValues(
			arguments["includeDeprecated"].(bool),
		)
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.Array{}
			for _, val := range *res {
				var json11 jsonw.Encodable = jsonw.Null
				if val != nil {
					json111 := ec.executeSelectionSet(sels, __EnumValueType{}, val)
					json11 = json111
				}
				json1 = append(json1, json11)
			}
			json = json1
		}
		return json
	
	case "inputFields":
		res := it.InputFields()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := jsonw.Array{}
			for _, val := range *res {
				var json11 jsonw.Encodable = jsonw.Null
				if val != nil {
					json111 := ec.executeSelectionSet(sels, __InputValueType{}, val)
					json11 = json111
				}
				json1 = append(json1, json11)
			}
			json = json1
		}
		return json
	
	case "ofType":
		res := it.OfType()
		var json jsonw.Encodable = jsonw.Null
		if res != nil {
			json1 := ec.executeSelectionSet(sels, __TypeType{}, res)
			json = json1
		}
		return json
	
	}
	panic("unknown field " + field)
}

var parsedSchema = schema.MustParse("\nschema {\n\tquery: Query\n\tmutation: Mutation\n}\n\ntype Query {\n\ttodo(id: Int!): Todo\n\tlastTodo: Todo\n\ttodos: [Todo!]!\n}\n\ntype Mutation {\n\tcreateTodo(text: String!): Todo!\n\tupdateTodo(id: Int!, done: Boolean!): Todo!\n}\n\ntype Todo {\n\tid: Int!\n\ttext: String!\n\tdone: Boolean!\n}\n")
