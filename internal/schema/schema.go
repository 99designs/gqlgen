package schema

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/lexer"
)

type Schema struct {
	EntryPoints map[string]NamedType
	Types       map[string]NamedType

	entryPointNames map[string]string
	objects         []*Object
	unions          []*Union
}

type NamedType interface {
	common.Type
	TypeName() string
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

func (*Scalar) Kind() string      { return "SCALAR" }
func (*Object) Kind() string      { return "OBJECT" }
func (*Interface) Kind() string   { return "INTERFACE" }
func (*Union) Kind() string       { return "UNION" }
func (*Enum) Kind() string        { return "ENUM" }
func (*InputObject) Kind() string { return "INPUT_OBJECT" }

func (t *Scalar) TypeName() string      { return t.Name }
func (t *Object) TypeName() string      { return t.Name }
func (t *Interface) TypeName() string   { return t.Name }
func (t *Union) TypeName() string       { return t.Name }
func (t *Enum) TypeName() string        { return t.Name }
func (t *InputObject) TypeName() string { return t.Name }

type Field struct {
	Name string
	Args InputObject
	Type common.Type
}

type InputValue struct {
	Name    string
	Type    common.Type
	Default interface{}
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

	for _, t := range s.Types {
		if err := resolveType(s, t); err != nil {
			return nil, err
		}
	}

	s.EntryPoints = make(map[string]NamedType)
	for key, name := range s.entryPointNames {
		t, ok := s.Types[name]
		if !ok {
			if !ok {
				return nil, errors.Errorf("type %q not found", name)
			}
		}
		s.EntryPoints[key] = t
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

func resolveType(s *Schema, t common.Type) *errors.GraphQLError {
	var err *errors.GraphQLError
	switch t := t.(type) {
	case *Scalar:
		// nothing
	case *Object:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *Interface:
		for _, f := range t.Fields {
			if err := resolveField(s, f); err != nil {
				return err
			}
		}
	case *Union:
		// nothing
	case *Enum:
		// nothing
	case *InputObject:
		for _, f := range t.InputFields {
			f.Type, err = resolveTypeName(s, f.Type)
			if err != nil {
				return err
			}
		}
	case *common.List:
		t.OfType, err = resolveTypeName(s, t.OfType)
		if err != nil {
			return err
		}
	case *common.NonNull:
		t.OfType, err = resolveTypeName(s, t.OfType)
		if err != nil {
			return err
		}
	default:
		panic("unreachable")
	}
	return nil
}

func resolveField(s *Schema, f *Field) *errors.GraphQLError {
	var err *errors.GraphQLError
	f.Type, err = resolveTypeName(s, f.Type)
	if err != nil {
		return err
	}
	resolveType(s, &f.Args)
	return nil
}

func resolveTypeName(s *Schema, t common.Type) (common.Type, *errors.GraphQLError) {
	if name, ok := t.(*common.TypeName); ok {
		refT, ok := s.Types[name.Name]
		if !ok {
			return nil, errors.Errorf("type %q not found", name.Name)
		}
		return refT, nil
	}
	if err := resolveType(s, t); err != nil {
		return nil, err
	}
	return t, nil
}

func parseSchema(l *lexer.Lexer) *Schema {
	s := &Schema{
		entryPointNames: make(map[string]string),
		Types: map[string]NamedType{
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
				s.entryPointNames[name] = typ
			}
			l.ConsumeToken('}')
		case "type":
			obj := parseObjectDecl(l)
			s.Types[obj.Name] = obj
			s.objects = append(s.objects, obj)
		case "interface":
			intf := parseInterfaceDecl(l)
			s.Types[intf.Name] = intf
		case "union":
			union := parseUnionDecl(l)
			s.Types[union.Name] = union
			s.unions = append(s.unions, union)
		case "enum":
			enum := parseEnumDecl(l)
			s.Types[enum.Name] = enum
		case "input":
			input := parseInputDecl(l)
			s.Types[input.Name] = input
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
		for {
			o.interfaceNames = append(o.interfaceNames, l.ConsumeIdent())
			if l.Peek() == '{' {
				break
			}
		}
	}
	l.ConsumeToken('{')
	o.Fields, o.FieldOrder = parseFields(l)
	l.ConsumeToken('}')
	return o
}

func parseInterfaceDecl(l *lexer.Lexer) *Interface {
	i := &Interface{}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	i.Fields, i.FieldOrder = parseFields(l)
	l.ConsumeToken('}')
	return i
}

func parseUnionDecl(l *lexer.Lexer) *Union {
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

func parseInputDecl(l *lexer.Lexer) *InputObject {
	i := &InputObject{
		InputFields: make(map[string]*InputValue),
	}
	i.Name = l.ConsumeIdent()
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		v := parseInputValue(l)
		i.InputFields[v.Name] = v
		i.InputFieldOrder = append(i.InputFieldOrder, v.Name)
	}
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

func parseFields(l *lexer.Lexer) (map[string]*Field, []string) {
	fields := make(map[string]*Field)
	var fieldOrder []string
	for l.Peek() != '}' {
		f := &Field{}
		f.Name = l.ConsumeIdent()
		if l.Peek() == '(' {
			f.Args.InputFields = make(map[string]*InputValue)
			l.ConsumeToken('(')
			for l.Peek() != ')' {
				v := parseInputValue(l)
				f.Args.InputFields[v.Name] = v
				f.Args.InputFieldOrder = append(f.Args.InputFieldOrder, v.Name)
			}
			l.ConsumeToken(')')
		}
		l.ConsumeToken(':')
		f.Type = common.ParseType(l)
		fields[f.Name] = f
		fieldOrder = append(fieldOrder, f.Name)
	}
	return fields, fieldOrder
}

func parseInputValue(l *lexer.Lexer) *InputValue {
	p := &InputValue{}
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	p.Type = common.ParseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = common.ParseValue(l, true).Eval(nil)
	}
	return p
}
