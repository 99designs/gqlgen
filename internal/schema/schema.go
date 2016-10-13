package schema

import (
	"errors"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Schema struct {
	Types map[string]*Object
}

type Type interface{}

type Scalar struct {
}

type TypeName struct {
	Name string
}

type Object struct {
	Fields map[string]Type
}

func Parse(schemaString string, filename string) (res *Schema, errRes error) {
	sc := &scanner.Scanner{}
	sc.Filename = filename
	sc.Init(strings.NewReader(schemaString))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(lexer.SyntaxError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	return parseSchema(lexer.New(sc)), nil
}

func parseSchema(l *lexer.Lexer) *Schema {
	s := &Schema{
		Types: make(map[string]*Object),
	}

	for l.Peek() != scanner.EOF {
		switch l.ConsumeIdent() {
		case "type":
			name, obj := parseTypeDecl(l)
			s.Types[name] = obj
		default:
			l.SyntaxError(`"type"`)
		}
	}

	return s
}

func parseTypeDecl(l *lexer.Lexer) (string, *Object) {
	typeName := l.ConsumeIdent()
	l.ConsumeToken('{')

	o := &Object{
		Fields: make(map[string]Type),
	}
	for l.Peek() != '}' {
		fieldName := l.ConsumeIdent()
		l.ConsumeToken(':')
		o.Fields[fieldName] = parseType(l)
	}
	l.ConsumeToken('}')

	return typeName, o
}

func parseType(l *lexer.Lexer) Type {
	// TODO check args
	// TODO check return type
	name := l.ConsumeIdent()
	if name == "String" {
		return &Scalar{}
	}
	return &TypeName{
		Name: name,
	}
}
