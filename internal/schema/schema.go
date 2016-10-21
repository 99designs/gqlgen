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

type Scalar struct {
	Name string
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

func (Scalar) isType()      {}
func (Object) isType()      {}
func (Interface) isType()   {}
func (Union) isType()       {}
func (Enum) isType()        {}
func (InputObject) isType() {}
func (List) isType()        {}
func (NonNull) isType()     {}

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

type typeRef struct {
	name   string
	target *Type
}

type context struct {
	typeRefs []*typeRef
}

func Parse(schemaString string) (s *Schema, err *errors.GraphQLError) {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(schemaString))

	l := lexer.New(sc)
	err = l.CatchSyntaxError(func() {
		c := &context{}
		s = parseSchema(l, c)
		for _, ref := range c.typeRefs {
			*ref.target = s.AllTypes[ref.name]
		}
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

func parseSchema(l *lexer.Lexer, c *context) *Schema {
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
			obj := parseObjectDecl(l, c)
			s.AllTypes[obj.Name] = obj
			s.Objects[obj.Name] = obj
		case "interface":
			intf := parseInterfaceDecl(l, c)
			s.AllTypes[intf.Name] = intf
			s.Interfaces[intf.Name] = intf
		case "union":
			union := parseUnionDecl(l, c)
			s.AllTypes[union.Name] = union
		case "enum":
			enum := parseEnumDecl(l, c)
			s.AllTypes[enum.Name] = enum
		case "input":
			input := parseInputDecl(l, c)
			s.AllTypes[input.Name] = input
		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "schema", "type", "enum", "interface", "union" or "input"`, x))
		}
	}

	return s
}

func parseObjectDecl(l *lexer.Lexer, c *context) *Object {
	o := &Object{}
	o.Name = l.ConsumeIdent()
	if l.Peek() == scanner.Ident {
		l.ConsumeKeyword("implements")
		o.Implements = l.ConsumeIdent()
	}
	l.ConsumeToken('{')
	o.Fields = parseFields(l, c)
	l.ConsumeToken('}')
	return o
}

func parseInterfaceDecl(l *lexer.Lexer, c *context) *Interface {
	i := &Interface{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	i.Fields = parseFields(l, c)
	l.ConsumeToken('}')
	return i
}

func parseUnionDecl(l *lexer.Lexer, c *context) *Union {
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

func parseInputDecl(l *lexer.Lexer, c *context) *InputObject {
	i := &InputObject{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	i.Fields = parseFields(l, c)
	l.ConsumeToken('}')
	return i
}

func parseEnumDecl(l *lexer.Lexer, c *context) *Enum {
	enum := &Enum{}
	enum.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		enum.Values = append(enum.Values, l.ConsumeIdent())
	}
	l.ConsumeToken('}')
	return enum
}

func parseFields(l *lexer.Lexer, c *context) map[string]*Field {
	fields := make(map[string]*Field)
	for l.Peek() != '}' {
		f := &Field{}
		f.Name = l.ConsumeIdent()
		if l.Peek() == '(' {
			f.Parameters = make(map[string]*Parameter)
			l.ConsumeToken('(')
			for l.Peek() != ')' {
				p := parseParameter(l, c)
				f.Parameters[p.Name] = p
			}
			l.ConsumeToken(')')
		}
		l.ConsumeToken(':')
		parseType(&f.Type, l, c)
		fields[f.Name] = f
	}
	return fields
}

func parseParameter(l *lexer.Lexer, c *context) *Parameter {
	p := &Parameter{}
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	parseType(&p.Type, l, c)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = l.ConsumeIdent()
	}
	return p
}

func parseType(target *Type, l *lexer.Lexer, c *context) {
	parseNonNil := func() {
		if l.Peek() == '!' {
			l.ConsumeToken('!')
			nn := &NonNull{}
			*target = nn
			target = &nn.Elem
		}
	}

	if l.Peek() == '[' {
		l.ConsumeToken('[')
		t := &List{}
		parseType(&t.Elem, l, c)
		l.ConsumeToken(']')
		parseNonNil()
		*target = t
		return
	}

	name := l.ConsumeIdent()
	parseNonNil()
	switch name {
	case "Int", "Float", "String", "Boolean", "ID":
		*target = &Scalar{Name: name}
	default:
		c.typeRefs = append(c.typeRefs, &typeRef{
			name:   name,
			target: target,
		})
	}
}
