package gen

import (
	"context"
	"fmt"
	"github.com/vektah/graphql-go/example/todo"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/schema"
	"strconv"
)

type Resolvers interface {
	Mutation_createTodo(ctx context.Context, text string) (todo.Todo, error)
	Mutation_updateTodo(ctx context.Context, id int, done bool) (todo.Todo, error)
	Query_todo(ctx context.Context, id int) (*todo.Todo, error)
	Query_lastTodo(ctx context.Context) (*todo.Todo, error)
	Query_todos(ctx context.Context) ([]todo.Todo, error)
}

var (
	mutationSatisfies     = []string{"Mutation"}
	querySatisfies        = []string{"Query"}
	todoSatisfies         = []string{"Todo"}
	__DirectiveSatisfies  = []string{"__Directive"}
	__EnumValueSatisfies  = []string{"__EnumValue"}
	__FieldSatisfies      = []string{"__Field"}
	__InputValueSatisfies = []string{"__InputValue"}
	__SchemaSatisfies     = []string{"__Schema"}
	__TypeSatisfies       = []string{"__Type"}
)

func _mutation(ec *executionContext, sel []query.Selection, it *interface{}) {
	groupedFieldSet := ec.collectFields(sel, mutationSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "createTodo":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Mutation_createTodo(
				ec.ctx,
				field.Args["text"].(string),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			_todo(ec, field.Selections, &res)
			continue

		case "updateTodo":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Mutation_updateTodo(
				ec.ctx,
				field.Args["id"].(int),
				field.Args["done"].(bool),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			_todo(ec, field.Selections, &res)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _query(ec *executionContext, sel []query.Selection, it *interface{}) {
	groupedFieldSet := ec.collectFields(sel, querySatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "todo":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_todo(
				ec.ctx,
				field.Args["id"].(int),
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			if res == nil {
				ec.json.Null()
			} else {
				_todo(ec, field.Selections, res)
			}
			continue

		case "lastTodo":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_lastTodo(
				ec.ctx,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			if res == nil {
				ec.json.Null()
			} else {
				_todo(ec, field.Selections, res)
			}
			continue

		case "todos":
			ec.json.ObjectKey(field.Alias)
			res, err := ec.resolvers.Query_todos(
				ec.ctx,
			)
			if err != nil {
				ec.Error(err)
				continue
			}
			ec.json.BeginArray()
			for _, val := range res {
				_todo(ec, field.Selections, &val)
			}
			ec.json.EndArray()
			continue

		case "__schema":
			ec.json.ObjectKey(field.Alias)
			res := ec.introspectSchema()
			if res == nil {
				ec.json.Null()
			} else {
				___Schema(ec, field.Selections, res)
			}
			continue

		case "__type":
			ec.json.ObjectKey(field.Alias)
			res := ec.introspectType(
				field.Args["name"].(string),
			)
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func _todo(ec *executionContext, sel []query.Selection, it *todo.Todo) {
	groupedFieldSet := ec.collectFields(sel, todoSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "id":
			ec.json.ObjectKey(field.Alias)
			res := it.ID
			ec.json.Int(res)
			continue

		case "text":
			ec.json.ObjectKey(field.Alias)
			res := it.Text
			ec.json.String(res)
			continue

		case "done":
			ec.json.ObjectKey(field.Alias)
			res := it.Done
			ec.json.Bool(res)
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Directive(ec *executionContext, sel []query.Selection, it *introspection.Directive) {
	groupedFieldSet := ec.collectFields(sel, __DirectiveSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "locations":
			ec.json.ObjectKey(field.Alias)
			res := it.Locations()
			ec.json.BeginArray()
			for _, val := range res {
				ec.json.String(val)
			}
			ec.json.EndArray()
			continue

		case "args":
			ec.json.ObjectKey(field.Alias)
			res := it.Args()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___InputValue(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___EnumValue(ec *executionContext, sel []query.Selection, it *introspection.EnumValue) {
	groupedFieldSet := ec.collectFields(sel, __EnumValueSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "isDeprecated":
			ec.json.ObjectKey(field.Alias)
			res := it.IsDeprecated()
			ec.json.Bool(res)
			continue

		case "deprecationReason":
			ec.json.ObjectKey(field.Alias)
			res := it.DeprecationReason()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Field(ec *executionContext, sel []query.Selection, it *introspection.Field) {
	groupedFieldSet := ec.collectFields(sel, __FieldSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "args":
			ec.json.ObjectKey(field.Alias)
			res := it.Args()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___InputValue(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		case "type":
			ec.json.ObjectKey(field.Alias)
			res := it.Type()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "isDeprecated":
			ec.json.ObjectKey(field.Alias)
			res := it.IsDeprecated()
			ec.json.Bool(res)
			continue

		case "deprecationReason":
			ec.json.ObjectKey(field.Alias)
			res := it.DeprecationReason()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___InputValue(ec *executionContext, sel []query.Selection, it *introspection.InputValue) {
	groupedFieldSet := ec.collectFields(sel, __InputValueSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			ec.json.String(res)
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "type":
			ec.json.ObjectKey(field.Alias)
			res := it.Type()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "defaultValue":
			ec.json.ObjectKey(field.Alias)
			res := it.DefaultValue()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Schema(ec *executionContext, sel []query.Selection, it *introspection.Schema) {
	groupedFieldSet := ec.collectFields(sel, __SchemaSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "types":
			ec.json.ObjectKey(field.Alias)
			res := it.Types()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___Type(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		case "queryType":
			ec.json.ObjectKey(field.Alias)
			res := it.QueryType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "mutationType":
			ec.json.ObjectKey(field.Alias)
			res := it.MutationType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "subscriptionType":
			ec.json.ObjectKey(field.Alias)
			res := it.SubscriptionType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		case "directives":
			ec.json.ObjectKey(field.Alias)
			res := it.Directives()
			ec.json.BeginArray()
			for _, val := range res {
				if val == nil {
					ec.json.Null()
				} else {
					___Directive(ec, field.Selections, val)
				}
			}
			ec.json.EndArray()
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

func ___Type(ec *executionContext, sel []query.Selection, it *introspection.Type) {
	groupedFieldSet := ec.collectFields(sel, __TypeSatisfies, map[string]bool{})
	ec.json.BeginObject()
	for _, field := range groupedFieldSet {
		switch field.Name {
		case "kind":
			ec.json.ObjectKey(field.Alias)
			res := it.Kind()
			ec.json.String(res)
			continue

		case "name":
			ec.json.ObjectKey(field.Alias)
			res := it.Name()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "description":
			ec.json.ObjectKey(field.Alias)
			res := it.Description()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.String(*res)
			}
			continue

		case "fields":
			ec.json.ObjectKey(field.Alias)
			res := it.Fields(
				field.Args["includeDeprecated"].(bool),
			)
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___Field(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "interfaces":
			ec.json.ObjectKey(field.Alias)
			res := it.Interfaces()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___Type(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "possibleTypes":
			ec.json.ObjectKey(field.Alias)
			res := it.PossibleTypes()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___Type(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "enumValues":
			ec.json.ObjectKey(field.Alias)
			res := it.EnumValues(
				field.Args["includeDeprecated"].(bool),
			)
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___EnumValue(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "inputFields":
			ec.json.ObjectKey(field.Alias)
			res := it.InputFields()
			if res == nil {
				ec.json.Null()
			} else {
				ec.json.BeginArray()
				for _, val := range *res {
					if val == nil {
						ec.json.Null()
					} else {
						___InputValue(ec, field.Selections, val)
					}
				}
				ec.json.EndArray()
			}
			continue

		case "ofType":
			ec.json.ObjectKey(field.Alias)
			res := it.OfType()
			if res == nil {
				ec.json.Null()
			} else {
				___Type(ec, field.Selections, res)
			}
			continue

		}
		panic("unknown field " + strconv.Quote(field.Name))
	}
	ec.json.EndObject()
}

var parsedSchema = schema.MustParse("\nschema {\n\tquery: Query\n\tmutation: Mutation\n}\n\ntype Query {\n\ttodo(id: Int!): Todo\n\tlastTodo: Todo\n\ttodos: [Todo!]!\n}\n\ntype Mutation {\n\tcreateTodo(text: String!): Todo!\n\tupdateTodo(id: Int!, done: Boolean!): Todo!\n}\n\ntype Todo {\n\tid: Int!\n\ttext: String!\n\tdone: Boolean!\n}\n")
var _ = fmt.Print
