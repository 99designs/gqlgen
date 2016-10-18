package schema

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Schema struct {
	EntryPoints map[string]string
	Types       map[string]Type
}

type Type interface {
	isType()
}

type Object struct {
	Name       string
	Implements string
	Fields     map[string]*Field
}

type Union struct {
	Name  string
	Types []string
}

type Enum struct {
	Name   string
	Values []string
}

type List struct {
	Elem Type
}

type TypeReference struct {
	Name string
}

func (Object) isType()        {}
func (Union) isType()         {}
func (Enum) isType()          {}
func (List) isType()          {}
func (TypeReference) isType() {}

type Field struct {
	Name       string
	Parameters map[string]*Parameter
	Type       Type
}

type Parameter struct {
	Name    string
	Type    string
	Default string
}

func Parse(schemaString string, filename string) (res *Schema, errRes error) {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanFloats | scanner.ScanStrings,
	}
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
		Types:       make(map[string]Type),
	}

	for l.Peek() != scanner.EOF {
		switch x := l.ConsumeIdent(); x {
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
			obj := parseTypeDecl(l)
			s.Types[obj.Name] = obj
		case "interface":
			obj := parseTypeDecl(l) // TODO
			s.Types[obj.Name] = obj
		case "union":
			union := parseUnionDecl(l)
			s.Types[union.Name] = union
		case "enum":
			enum := parseEnumDecl(l)
			s.Types[enum.Name] = enum
		case "input":
			parseInputDecl(l) // TODO
		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union" or "input"`, x))
		}
	}

	return s
}

func parseTypeDecl(l *lexer.Lexer) *Object {
	o := &Object{
		Fields: make(map[string]*Field),
	}

	o.Name = l.ConsumeIdent()
	if l.Peek() == scanner.Ident {
		l.ConsumeKeyword("implements")
		o.Implements = l.ConsumeIdent()
	}
	l.ConsumeToken('{')

	for l.Peek() != '}' {
		f := parseField(l)
		o.Fields[f.Name] = f
	}
	l.ConsumeToken('}')

	return o
}

func parseEnumDecl(l *lexer.Lexer) *Enum {
	enum := &Enum{}
	enum.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		enum.Values = append(enum.Values, l.ConsumeIdent())
	}
	l.ConsumeToken('}')
	return enum
}

func parseUnionDecl(l *lexer.Lexer) *Union {
	union := &Union{}
	union.Name = l.ConsumeIdent()
	l.ConsumeToken('=')
	union.Types = []string{l.ConsumeIdent()}
	for l.Peek() == '|' {
		l.ConsumeToken('|')
		union.Types = append(union.Types, l.ConsumeIdent())
	}
	return union
}

func parseInputDecl(l *lexer.Lexer) {
	l.ConsumeIdent()
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		parseField(l)
	}
	l.ConsumeToken('}')
}

func parseField(l *lexer.Lexer) *Field {
	f := &Field{}
	f.Name = l.ConsumeIdent()
	if l.Peek() == '(' {
		f.Parameters = make(map[string]*Parameter)
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			p := parseParameter(l)
			f.Parameters[p.Name] = p
		}
		l.ConsumeToken(')')
	}
	l.ConsumeToken(':')
	f.Type = parseType(l)
	if l.Peek() == '!' {
		l.ConsumeToken('!') // TODO
	}
	return f
}

func parseParameter(l *lexer.Lexer) *Parameter {
	p := &Parameter{}
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	p.Type = l.ConsumeIdent()
	if l.Peek() == '!' {
		l.ConsumeToken('!') // TODO
	}
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = l.ConsumeIdent()
	}
	return p
}

func parseType(l *lexer.Lexer) Type {
	if l.Peek() == '[' {
		return parseList(l)
	}

	return &TypeReference{
		Name: l.ConsumeIdent(),
	}
}

func parseList(l *lexer.Lexer) *List {
	l.ConsumeToken('[')
	elem := parseType(l)
	l.ConsumeToken(']')
	return &List{Elem: elem}
}
