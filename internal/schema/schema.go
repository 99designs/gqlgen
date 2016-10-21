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
	Types       map[string]Type

	objects []*Object
	unions  []*Union
}

type Type interface {
	isType()
}

type Scalar struct {
	Name string
}

type Object struct {
	Name       string
	Interfaces []*Interface
	Fields     map[string]*Field
	FieldOrder []string

	interfaceNames []string
}

type Interface struct {
	Name          string
	PossibleTypes []*Object
	Fields        map[string]*Field
	FieldOrder    []string
}

type Union struct {
	Name          string
	PossibleTypes []*Object

	typeNames []string
}

type Enum struct {
	Name   string
	Values []string
}

type InputObject struct {
	Name            string
	InputFields     map[string]*InputValue
	InputFieldOrder []string
}

type List struct {
	OfType Type
}

type NonNull struct {
	OfType Type
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
	Name     string
	Args     map[string]*InputValue
	ArgOrder []string
	Type     Type
}

type InputValue struct {
	Name    string
	Type    Type
	Default interface{}
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

	c := &context{}
	l := lexer.New(sc)
	err = l.CatchSyntaxError(func() {
		s = parseSchema(l, c)
	})
	if err != nil {
		return nil, err
	}

	for _, ref := range c.typeRefs {
		t, ok := s.Types[ref.name]
		if !ok {
			return nil, errors.Errorf("type %q not found", ref.name)
		}
		*ref.target = t
	}

	for _, obj := range s.objects {
		obj.Interfaces = make([]*Interface, len(obj.interfaceNames))
		for i, intfName := range obj.interfaceNames {
			t, ok := s.Types[intfName]
			if !ok {
				return nil, errors.Errorf("interface %q not found", intfName)
			}
			intf, ok := t.(*Interface)
			if !ok {
				return nil, errors.Errorf("type %q is not an interface", intfName)
			}
			obj.Interfaces[i] = intf
			intf.PossibleTypes = append(intf.PossibleTypes, obj)
		}
	}

	for _, union := range s.unions {
		union.PossibleTypes = make([]*Object, len(union.typeNames))
		for i, name := range union.typeNames {
			t, ok := s.Types[name]
			if !ok {
				return nil, errors.Errorf("object type %q not found", name)
			}
			obj, ok := t.(*Object)
			if !ok {
				return nil, errors.Errorf("type %q is not an object", name)
			}
			union.PossibleTypes[i] = obj
		}
	}

	return s, nil
}

func parseSchema(l *lexer.Lexer, c *context) *Schema {
	s := &Schema{
		EntryPoints: make(map[string]string),
		Types: map[string]Type{
			"Int":     &Scalar{Name: "Int"},
			"Float":   &Scalar{Name: "Float"},
			"String":  &Scalar{Name: "String"},
			"Boolean": &Scalar{Name: "Boolean"},
			"ID":      &Scalar{Name: "ID"},
		},
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
			s.Types[obj.Name] = obj
			s.objects = append(s.objects, obj)
		case "interface":
			intf := parseInterfaceDecl(l, c)
			s.Types[intf.Name] = intf
		case "union":
			union := parseUnionDecl(l, c)
			s.Types[union.Name] = union
			s.unions = append(s.unions, union)
		case "enum":
			enum := parseEnumDecl(l, c)
			s.Types[enum.Name] = enum
		case "input":
			input := parseInputDecl(l, c)
			s.Types[input.Name] = input
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
		for {
			o.interfaceNames = append(o.interfaceNames, l.ConsumeIdent())
			if l.Peek() == '{' {
				break
			}
		}
	}
	l.ConsumeToken('{')
	o.Fields, o.FieldOrder = parseFields(l, c)
	l.ConsumeToken('}')
	return o
}

func parseInterfaceDecl(l *lexer.Lexer, c *context) *Interface {
	i := &Interface{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	i.Fields, i.FieldOrder = parseFields(l, c)
	l.ConsumeToken('}')
	return i
}

func parseUnionDecl(l *lexer.Lexer, c *context) *Union {
	union := &Union{}
	union.Name = l.ConsumeIdent()
	l.ConsumeToken('=')
	union.typeNames = []string{l.ConsumeIdent()}
	for l.Peek() == '|' {
		l.ConsumeToken('|')
		union.typeNames = append(union.typeNames, l.ConsumeIdent())
	}
	return union
}

func parseInputDecl(l *lexer.Lexer, c *context) *InputObject {
	i := &InputObject{
		InputFields: make(map[string]*InputValue),
	}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		v := parseInputValue(l, c)
		i.InputFields[v.Name] = v
		i.InputFieldOrder = append(i.InputFieldOrder, v.Name)
	}
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

func parseFields(l *lexer.Lexer, c *context) (map[string]*Field, []string) {
	fields := make(map[string]*Field)
	var fieldOrder []string
	for l.Peek() != '}' {
		f := &Field{}
		f.Name = l.ConsumeIdent()
		if l.Peek() == '(' {
			f.Args = make(map[string]*InputValue)
			l.ConsumeToken('(')
			for l.Peek() != ')' {
				v := parseInputValue(l, c)
				f.Args[v.Name] = v
				f.ArgOrder = append(f.ArgOrder, v.Name)
			}
			l.ConsumeToken(')')
		}
		l.ConsumeToken(':')
		parseType(&f.Type, l, c)
		fields[f.Name] = f
		fieldOrder = append(fieldOrder, f.Name)
	}
	return fields, fieldOrder
}

func parseInputValue(l *lexer.Lexer, c *context) *InputValue {
	p := &InputValue{}
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	parseType(&p.Type, l, c)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = parseValue(l)
	}
	return p
}

func parseType(target *Type, l *lexer.Lexer, c *context) {
	parseNonNil := func() {
		if l.Peek() == '!' {
			l.ConsumeToken('!')
			nn := &NonNull{}
			*target = nn
			target = &nn.OfType
		}
	}

	if l.Peek() == '[' {
		l.ConsumeToken('[')
		t := &List{}
		parseType(&t.OfType, l, c)
		l.ConsumeToken(']')
		parseNonNil()
		*target = t
		return
	}

	name := l.ConsumeIdent()
	parseNonNil()
	c.typeRefs = append(c.typeRefs, &typeRef{
		name:   name,
		target: target,
	})
}

func parseValue(l *lexer.Lexer) interface{} {
	switch l.Peek() {
	case scanner.Int:
		return l.ConsumeInt()
	case scanner.Float:
		return l.ConsumeFloat()
	case scanner.String:
		return l.ConsumeString()
	case scanner.Ident:
		switch ident := l.ConsumeIdent(); ident {
		case "true":
			return true
		case "false":
			return false
		default:
			return ident
		}
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
