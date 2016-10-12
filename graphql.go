package graphql

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"text/scanner"
)

type Schema struct {
	types    map[string]*object
	resolver reflect.Value
}

type typ interface {
	exec(schema *Schema, sel *selectionSet, resolver reflect.Value) interface{}
}

type scalar struct {
}

type typeName struct {
	name string
}

type object struct {
	fields map[string]typ
}

type parseError string

func NewSchema(schema string, filename string, resolver interface{}) (res *Schema, errRes error) {
	sc := &scanner.Scanner{}
	sc.Filename = filename
	sc.Init(strings.NewReader(schema))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(parseError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	s := parseSchema(newLexer(sc))
	s.resolver = reflect.ValueOf(resolver)
	// TODO type check resolver
	return s, nil
}

func parseSchema(l *lexer) *Schema {
	s := &Schema{
		types: make(map[string]*object),
	}

	for l.peek() != scanner.EOF {
		switch l.consumeIdent() {
		case "type":
			name, obj := parseTypeDecl(l)
			s.types[name] = obj
		default:
			l.syntaxError(`"type"`)
		}
	}

	return s
}

func parseTypeDecl(l *lexer) (string, *object) {
	typeName := l.consumeIdent()
	l.consumeToken('{')

	o := &object{
		fields: make(map[string]typ),
	}
	for l.peek() != '}' {
		fieldName := l.consumeIdent()
		l.consumeToken(':')
		o.fields[fieldName] = parseType(l)
	}
	l.consumeToken('}')

	return typeName, o
}

func parseType(l *lexer) typ {
	// TODO check args
	// TODO check return type
	name := l.consumeIdent()
	if name == "String" {
		return &scalar{}
	}
	return &typeName{
		name: name,
	}
}

func (s *Schema) Exec(query string) (res []byte, errRes error) {
	sc := &scanner.Scanner{}
	sc.Init(strings.NewReader(query))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(parseError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	rawRes := s.types["Query"].exec(s, parseSelectionSet(newLexer(sc)), s.resolver)
	return json.Marshal(rawRes)
}

type selectionSet struct {
	selections []*field
}

func parseSelectionSet(l *lexer) *selectionSet {
	sel := &selectionSet{}
	l.consumeToken('{')
	for l.peek() != '}' {
		sel.selections = append(sel.selections, parseField(l))
	}
	l.consumeToken('}')
	return sel
}

type field struct {
	name string
	sel  *selectionSet
}

func parseField(l *lexer) *field {
	f := &field{}
	f.name = l.consumeIdent()
	if l.peek() == '{' {
		f.sel = parseSelectionSet(l)
	}
	return f
}

func (o *object) exec(schema *Schema, sel *selectionSet, resolver reflect.Value) interface{} {
	res := make(map[string]interface{})
	for _, f := range sel.selections {
		m := findMethod(resolver.Type(), f.name)
		res[f.name] = o.fields[f.name].exec(schema, f.sel, resolver.Method(m).Call(nil)[0])
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

func (s *scalar) exec(schema *Schema, sel *selectionSet, resolver reflect.Value) interface{} {
	if !resolver.IsValid() {
		return "bad"
	}
	return resolver.Interface()
}

func (s *typeName) exec(schema *Schema, sel *selectionSet, resolver reflect.Value) interface{} {
	return schema.types[s.name].exec(schema, sel, resolver)
}
