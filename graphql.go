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
			if skipByDirective(r, sel.Directives) {
				continue
			}
			execField(s, r, t, sel, resolver, result)
		case *query.FragmentSpread:
			if skipByDirective(r, sel.Directives) {
				continue
			}
			execSelectionSet(s, r, t, r.Fragments[sel.Name].SelSet, resolver, result)
		default:
			panic("invalid type")
		}
	}
}

func execField(s *Schema, r *request, t *schema.Object, f *query.Field, resolver reflect.Value, result map[string]interface{}) {
	sf := t.Fields[f.Name]
	m := resolver.Method(findMethod(resolver.Type(), f.Name))
	var in []reflect.Value
	if len(sf.Parameters) != 0 {
		args := reflect.New(m.Type().In(0))
		for name, param := range sf.Parameters {
			value, ok := f.Arguments[name]
			if !ok {
				value = &query.Literal{Value: param.Default}
			}
			rf := args.Elem().FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, name) })
			rf.Set(reflect.ValueOf(execValue(r, value)))
		}
		in = []reflect.Value{args.Elem()}
	}
	result[f.Alias] = exec(s, r, sf.Type, f.SelSet, m.Call(in)[0])
}

func skipByDirective(r *request, d map[string]*query.Directive) bool {
	if skip, ok := d["skip"]; ok {
		if execValue(r, skip.Arguments["if"]).(bool) {
			return true
		}
	}
	if include, ok := d["include"]; ok {
		if !execValue(r, include.Arguments["if"]).(bool) {
			return true
		}
	}
	return false
}

func execValue(r *request, v query.Value) interface{} {
	switch v := v.(type) {
	case *query.Variable:
		return r.Variables[v.Name]
	case *query.Literal:
		return v.Value
	default:
		panic("invalid value")
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
