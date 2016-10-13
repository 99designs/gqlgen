package schema

import (
	"errors"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Schema struct {
	EntryPoints map[string]string
	Types       map[string]*Object
}

type Type interface{}

type Scalar struct {
}

type Array struct {
	Elem Type
}

type TypeName struct {
	Name string
}

type Object struct {
	Fields map[string]*Field
}

type Field struct {
	Name       string
	Parameters map[string]string
	Type       Type
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
		EntryPoints: make(map[string]string),
		Types:       make(map[string]*Object),
	}

	for l.Peek() != scanner.EOF {
		switch l.ConsumeIdent() {
		case "schema":
			l.ConsumeToken('{')
			for l.Peek() != '}' {
				name := l.ConsumeIdent()
				l.ConsumeToken(':')
				typ := l.ConsumeIdent()
				s.EntryPoints[name] = typ
			}
			l.ConsumeToken('}')
		case "type":
			name, obj := parseTypeDecl(l)
			s.Types[name] = obj
		default:
			l.UnexpectedSyntaxError(`"schema" or "type"`)
		}
	}

	return s
}

func parseTypeDecl(l *lexer.Lexer) (string, *Object) {
	typeName := l.ConsumeIdent()
	l.ConsumeToken('{')

	o := &Object{
		Fields: make(map[string]*Field),
	}
	for l.Peek() != '}' {
		f := parseField(l)
		o.Fields[f.Name] = f
	}
	l.ConsumeToken('}')

	return typeName, o
}

func parseField(l *lexer.Lexer) *Field {
	f := &Field{
		Parameters: make(map[string]string),
	}
	f.Name = l.ConsumeIdent()
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		if l.Peek() != ')' {
			name, typ := parseParameter(l)
			f.Parameters[name] = typ
			for l.Peek() != ')' {
				l.ConsumeToken(',')
				name, typ := parseParameter(l)
				f.Parameters[name] = typ
			}
		}
		l.ConsumeToken(')')
	}
	l.ConsumeToken(':')
	f.Type = parseType(l)
	return f
}

func parseParameter(l *lexer.Lexer) (string, string) {
	name := l.ConsumeIdent()
	l.ConsumeToken(':')
	typ := l.ConsumeIdent()
	return name, typ
}

func parseType(l *lexer.Lexer) Type {
	if l.Peek() == '[' {
		return parseArray(l)
	}

	name := l.ConsumeIdent()
	if name == "String" || name == "Float" {
		return &Scalar{}
	}
	return &TypeName{
		Name: name,
	}
}

func parseArray(l *lexer.Lexer) *Array {
	l.ConsumeToken('[')
	elem := parseType(l)
	l.ConsumeToken(']')
	return &Array{Elem: elem}
}
