package graphql

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	*schema.Schema
	resolver reflect.Value
}

func NewSchema(schemaString string, filename string, resolver interface{}) (res *Schema, errRes error) {
	s, err := schema.Parse(schemaString, filename)
	if err != nil {
		return nil, err
	}

	// TODO type check resolver
	return &Schema{
		Schema:   s,
		resolver: reflect.ValueOf(resolver),
	}, nil
}

func (s *Schema) Exec(queryString string) (res []byte, errRes error) {
	q, err := query.Parse(queryString)
	if err != nil {
		return nil, err
	}

	rawRes := exec(s, q, s.Types[s.EntryPoints["query"]], q.Root, s.resolver)
	return json.Marshal(rawRes)
}

func exec(s *Schema, q *query.Query, t schema.Type, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	switch t := t.(type) {
	case *schema.Scalar:
		return resolver.Interface()

	case *schema.Object:
		result := make(map[string]interface{})
		execSelectionSet(s, q, t, selSet, resolver, result)
		return result

	case *schema.Enum:
		return resolver.Interface()

	case *schema.List:
		a := make([]interface{}, resolver.Len())
		for i := range a {
			a[i] = exec(s, q, t.Elem, selSet, resolver.Index(i))
		}
		return a

	case *schema.TypeReference:
		return exec(s, q, s.Types[t.Name], selSet, resolver)

	default:
		panic("invalid type")
	}
}

func execSelectionSet(s *Schema, q *query.Query, t *schema.Object, selSet *query.SelectionSet, resolver reflect.Value, result map[string]interface{}) {
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			sf := t.Fields[sel.Name]
			m := resolver.Method(findMethod(resolver.Type(), sel.Name))
			var in []reflect.Value
			if len(sf.Parameters) != 0 {
				args := reflect.New(m.Type().In(0))
				for name, param := range sf.Parameters {
					value, ok := sel.Arguments[name]
					if !ok {
						value = &query.Value{Value: param.Default}
					}
					rf := args.Elem().FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, name) })
					rf.Set(reflect.ValueOf(value.Value))
				}
				in = []reflect.Value{args.Elem()}
			}
			result[sel.Alias] = exec(s, q, sf.Type, sel.SelSet, m.Call(in)[0])

		case *query.FragmentSpread:
			execSelectionSet(s, q, t, q.Fragments[sel.Name].SelSet, resolver, result)

		default:
			panic("invalid type")
		}
	}
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(name, t.Method(i).Name) {
			return i
		}
	}
	return -1
}
