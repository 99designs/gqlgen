package schema

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/lexer"
)

type Schema struct {
	EntryPoints map[string]string
	AllTypes    map[string]Type
	Objects     map[string]*Object
	Interfaces  map[string]*Interface
}

type Type interface {
	isType()
}

type Object struct {
	Name       string
	Implements string
	Fields     map[string]*Field
}

type Interface struct {
	Name          string
	ImplementedBy []string
	Fields        map[string]*Field
}

type Union struct {
	Name  string
	Types []string
}

type Enum struct {
	Name   string
	Values []string
}

type InputObject struct {
	Name   string
	Fields map[string]*Field
}

type List struct {
	Elem Type
}

type NonNull struct {
	Elem Type
}

type TypeReference struct {
	Name string
}

func (Object) isType()        {}
func (Interface) isType()     {}
func (Union) isType()         {}
func (Enum) isType()          {}
func (InputObject) isType()   {}
func (List) isType()          {}
func (NonNull) isType()       {}
func (TypeReference) isType() {}

type Field struct {
	Name       string
	Parameters map[string]*Parameter
	Type       Type
}

type Parameter struct {
	Name    string
	Type    Type
	Default string
}

func Parse(schemaString string) (s *Schema, err *errors.GraphQLError) {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(schemaString))

	l := lexer.New(sc)
	err = l.CatchSyntaxError(func() {
		s = parseSchema(l)
	})
	if err != nil {
		return nil, err
	}

	for _, obj := range s.Objects {
		if obj.Implements != "" {
			intf, ok := s.Interfaces[obj.Implements]
			if !ok {
				return nil, errors.Errorf("interface %q not found", obj.Implements)
			}
			intf.ImplementedBy = append(intf.ImplementedBy, obj.Name)
		}
	}

	return s, nil
}

func parseSchema(l *lexer.Lexer) *Schema {
	s := &Schema{
		EntryPoints: make(map[string]string),
		AllTypes:    make(map[string]Type),
		Objects:     make(map[string]*Object),
		Interfaces:  make(map[string]*Interface),
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
			obj := parseObjectDecl(l)
			s.AllTypes[obj.Name] = obj
			s.Objects[obj.Name] = obj
		case "interface":
			intf := parseInterfaceDecl(l)
			s.AllTypes[intf.Name] = intf
			s.Interfaces[intf.Name] = intf
		case "union":
			union := parseUnionDecl(l)
			s.AllTypes[union.Name] = union
		case "enum":
			enum := parseEnumDecl(l)
			s.AllTypes[enum.Name] = enum
		case "input":
			input := parseInputDecl(l)
			s.AllTypes[input.Name] = input
		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union" or "input"`, x))
		}
	}

	return s
}

func parseObjectDecl(l *lexer.Lexer) *Object {
	o := &Object{}
	o.Name = l.ConsumeIdent()
	if l.Peek() == scanner.Ident {
		l.ConsumeKeyword("implements")
		o.Implements = l.ConsumeIdent()
	}
	l.ConsumeToken('{')
	o.Fields = parseFields(l)
	l.ConsumeToken('}')
	return o
}

func parseInterfaceDecl(l *lexer.Lexer) *Interface {
	i := &Interface{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	i.Fields = parseFields(l)
	l.ConsumeToken('}')
	return i
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

func parseInputDecl(l *lexer.Lexer) *InputObject {
	i := &InputObject{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	i.Fields = parseFields(l)
	l.ConsumeToken('}')
	return i
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

func parseFields(l *lexer.Lexer) map[string]*Field {
	fields := make(map[string]*Field)
	for l.Peek() != '}' {
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
		fields[f.Name] = f
	}
	return fields
}

func parseParameter(l *lexer.Lexer) *Parameter {
	p := &Parameter{}
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	p.Type = parseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = l.ConsumeIdent()
	}
	return p
}

func parseType(l *lexer.Lexer) Type {
	t := parseNullableType(l)
	if l.Peek() == '!' {
		l.ConsumeToken('!')
		return &NonNull{t}
	}
	return t
}

func parseNullableType(l *lexer.Lexer) Type {
	if l.Peek() == '[' {
		l.ConsumeToken('[')
		elem := parseType(l)
		l.ConsumeToken(']')
		return &List{Elem: elem}
	}
	name := l.ConsumeIdent()
	return &TypeReference{
		Name: name,
	}
}
