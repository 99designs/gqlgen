package graphql

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
	"github.com/neelance/graphql-go/internal/query"
)

type Schema struct {
	types    map[string]*object
	resolver reflect.Value
}

type typ interface {
	exec(schema *Schema, sel *query.SelectionSet, resolver reflect.Value) interface{}
}

type scalar struct {
}

type typeName struct {
	name string
}

type object struct {
	fields map[string]typ
}

func NewSchema(schema string, filename string, resolver interface{}) (res *Schema, errRes error) {
	sc := &scanner.Scanner{}
	sc.Filename = filename
	sc.Init(strings.NewReader(schema))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(lexer.SyntaxError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	s := parseSchema(lexer.New(sc))
	s.resolver = reflect.ValueOf(resolver)
	// TODO type check resolver
	return s, nil
}

func parseSchema(l *lexer.Lexer) *Schema {
	s := &Schema{
		types: make(map[string]*object),
	}

	for l.Peek() != scanner.EOF {
		switch l.ConsumeIdent() {
		case "type":
			name, obj := parseTypeDecl(l)
			s.types[name] = obj
		default:
			l.SyntaxError(`"type"`)
		}
	}

	return s
}

func parseTypeDecl(l *lexer.Lexer) (string, *object) {
	typeName := l.ConsumeIdent()
	l.ConsumeToken('{')

	o := &object{
		fields: make(map[string]typ),
	}
	for l.Peek() != '}' {
		fieldName := l.ConsumeIdent()
		l.ConsumeToken(':')
		o.fields[fieldName] = parseType(l)
	}
	l.ConsumeToken('}')

	return typeName, o
}

func parseType(l *lexer.Lexer) typ {
	// TODO check args
	// TODO check return type
	name := l.ConsumeIdent()
	if name == "String" {
		return &scalar{}
	}
	return &typeName{
		name: name,
	}
}

func (s *Schema) Exec(queryInput string) (res []byte, errRes error) {
	sc := &scanner.Scanner{}
	sc.Init(strings.NewReader(queryInput))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(lexer.SyntaxError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	rawRes := s.types["Query"].exec(s, query.Parse(lexer.New(sc)), s.resolver)
	return json.Marshal(rawRes)
}

func (o *object) exec(schema *Schema, sel *query.SelectionSet, resolver reflect.Value) interface{} {
	res := make(map[string]interface{})
	for _, f := range sel.Selections {
		m := findMethod(resolver.Type(), f.Name)
		res[f.Name] = o.fields[f.Name].exec(schema, f.Sel, resolver.Method(m).Call(nil)[0])
	}
	return res
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(name, t.Method(i).Name) {
			return i
		}
	}
	return -1
}

func (s *scalar) exec(schema *Schema, sel *query.SelectionSet, resolver reflect.Value) interface{} {
	if !resolver.IsValid() {
		return "bad"
	}
	return resolver.Interface()
}

func (s *typeName) exec(schema *Schema, sel *query.SelectionSet, resolver reflect.Value) interface{} {
	return schema.types[s.name].exec(schema, sel, resolver)
}
