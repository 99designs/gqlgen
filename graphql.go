package graphql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	*schema.Schema
	resolver reflect.Value
}

type request struct {
	*query.Document
	Variables map[string]interface{}
}

func NewSchema(schemaString string, filename string, resolver interface{}) (*Schema, error) {
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

func (s *Schema) Exec(queryString string, operationName string, variables map[string]interface{}) ([]byte, error) {
	d, err := query.Parse(queryString)
	if err != nil {
		return nil, err
	}

	if operationName == "" && len(d.Operations) == 1 {
		for name := range d.Operations {
			operationName = name
		}
	}

	op, ok := d.Operations[operationName]
	if !ok {
		return nil, fmt.Errorf("no operation with name %q", operationName)
	}

	r := &request{Document: d, Variables: variables}
	rawRes := exec(s, r, s.Types[s.EntryPoints["query"]], op.SelSet, s.resolver)
	return json.Marshal(rawRes)
}

func exec(s *Schema, r *request, t schema.Type, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	switch t := t.(type) {
	case *schema.Scalar:
		return resolver.Interface()

	case *schema.Object:
		result := make(map[string]interface{})
		execSelectionSet(s, r, t, selSet, resolver, result)
		return result

	case *schema.Enum:
		return resolver.Interface()

	case *schema.List:
		a := make([]interface{}, resolver.Len())
		for i := range a {
			a[i] = exec(s, r, t.Elem, selSet, resolver.Index(i))
		}
		return a

	case *schema.TypeReference:
		return exec(s, r, s.Types[t.Name], selSet, resolver)

	default:
		panic("invalid type")
	}
}

func execSelectionSet(s *Schema, r *request, t *schema.Object, selSet *query.SelectionSet, resolver reflect.Value, result map[string]interface{}) {
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
						value = &query.Literal{Value: param.Default}
					}
					rf := args.Elem().FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, name) })
					switch v := value.(type) {
					case *query.Variable:
						rf.Set(reflect.ValueOf(r.Variables[v.Name]))
					case *query.Literal:
						rf.Set(reflect.ValueOf(v.Value))
					default:
						panic("invalid value")
					}
				}
				in = []reflect.Value{args.Elem()}
			}
			result[sel.Alias] = exec(s, r, sf.Type, sel.SelSet, m.Call(in)[0])

		case *query.FragmentSpread:
			execSelectionSet(s, r, t, r.Fragments[sel.Name].SelSet, resolver, result)

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
