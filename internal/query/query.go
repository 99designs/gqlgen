package query

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/lexer"
)

type Document struct {
	Operations map[string]*Operation
	Fragments  map[string]*NamedFragment
}

type Operation struct {
	Type      OperationType
	Name      string
	Variables map[string]*VariableDef
	SelSet    *SelectionSet
}

type OperationType int

const (
	Query OperationType = iota
	Mutation
)

type VariableDef struct {
	Name string
	Type string
}

type NamedFragment struct {
	Fragment
	Name string
}

type Fragment struct {
	On     string
	SelSet *SelectionSet
}

type SelectionSet struct {
	Selections []Selection
}

type Selection interface {
	isSelection()
}

type Field struct {
	Alias      string
	Name       string
	Arguments  map[string]Value
	Directives map[string]*Directive
	SelSet     *SelectionSet
}

type Directive struct {
	Name      string
	Arguments map[string]Value
}

type FragmentSpread struct {
	Name       string
	Directives map[string]*Directive
}

type InlineFragment struct {
	Fragment
	Directives map[string]*Directive
}

func (Field) isSelection()          {}
func (FragmentSpread) isSelection() {}
func (InlineFragment) isSelection() {}

type Value interface {
	isValue()
}

type Variable struct {
	Name string
}

type Literal struct {
	Value interface{}
}

func (Variable) isValue() {}
func (Literal) isValue()  {}

func Parse(queryString string) (doc *Document, err *errors.GraphQLError) {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(queryString))

	l := lexer.New(sc)
	err = l.CatchSyntaxError(func() {
		doc = parseDocument(l)
	})
	return
}

func parseDocument(l *lexer.Lexer) *Document {
	d := &Document{
		Operations: make(map[string]*Operation),
		Fragments:  make(map[string]*NamedFragment),
	}
	for l.Peek() != scanner.EOF {
		if l.Peek() == '{' {
			d.Operations[""] = &Operation{SelSet: parseSelectionSet(l)}
			continue
		}

		switch x := l.ConsumeIdent(); x {
		case "query":
			q := parseOperation(l, Query)
			d.Operations[q.Name] = q

		case "mutation":
			q := parseOperation(l, Mutation)
			d.Operations[q.Name] = q

		case "fragment":
			f := parseFragment(l)
			d.Fragments[f.Name] = f

		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "fragment"`, x))
		}
	}
	return d
}

func parseOperation(l *lexer.Lexer, opType OperationType) *Operation {
	op := &Operation{Type: opType}
	if l.Peek() == scanner.Ident {
		op.Name = l.ConsumeIdent()
	}
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		op.Variables = make(map[string]*VariableDef)
		for l.Peek() != ')' {
			v := parseVariableDef(l)
			op.Variables[v.Name] = v
		}
		l.ConsumeToken(')')
	}
	op.SelSet = parseSelectionSet(l)
	return op
}

func parseFragment(l *lexer.Lexer) *NamedFragment {
	f := &NamedFragment{}
	f.Name = l.ConsumeIdent()
	l.ConsumeKeyword("on")
	f.On = l.ConsumeIdent()
	f.SelSet = parseSelectionSet(l)
	return f
}

func parseVariableDef(l *lexer.Lexer) *VariableDef {
	v := &VariableDef{}
	l.ConsumeToken('$')
	v.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	v.Type = l.ConsumeIdent()
	if l.Peek() == '!' {
		l.ConsumeToken('!') // TODO
	}
	return v
}

func parseSelectionSet(l *lexer.Lexer) *SelectionSet {
	sel := &SelectionSet{}
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		sel.Selections = append(sel.Selections, parseSelection(l))
	}
	l.ConsumeToken('}')
	return sel
}

func parseSelection(l *lexer.Lexer) Selection {
	if l.Peek() == '.' {
		return parseSpread(l)
	}
	return parseField(l)
}

func parseField(l *lexer.Lexer) *Field {
	f := &Field{
		Directives: make(map[string]*Directive),
	}
	f.Alias = l.ConsumeIdent()
	f.Name = f.Alias
	if l.Peek() == ':' {
		l.ConsumeToken(':')
		f.Name = l.ConsumeIdent()
	}
	if l.Peek() == '(' {
		f.Arguments = parseArguments(l)
	}
	for l.Peek() == '@' {
		d := parseDirective(l)
		f.Directives[d.Name] = d
	}
	if l.Peek() == '{' {
		f.SelSet = parseSelectionSet(l)
	}
	return f
}

func parseArguments(l *lexer.Lexer) map[string]Value {
	args := make(map[string]Value)
	l.ConsumeToken('(')
	if l.Peek() != ')' {
		name, value := parseArgument(l)
		args[name] = value
		for l.Peek() != ')' {
			l.ConsumeToken(',')
			name, value := parseArgument(l)
			args[name] = value
		}
	}
	l.ConsumeToken(')')
	return args
}

func parseDirective(l *lexer.Lexer) *Directive {
	d := &Directive{}
	l.ConsumeToken('@')
	d.Name = l.ConsumeIdent()
	if l.Peek() == '(' {
		d.Arguments = parseArguments(l)
	}
	return d
}

func parseSpread(l *lexer.Lexer) Selection {
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	ident := l.ConsumeIdent()

	if ident == "on" {
		f := &InlineFragment{
			Directives: make(map[string]*Directive),
		}
		f.On = l.ConsumeIdent()
		for l.Peek() == '@' {
			d := parseDirective(l)
			f.Directives[d.Name] = d
		}
		f.SelSet = parseSelectionSet(l)
		return f
	}

	fs := &FragmentSpread{
		Directives: make(map[string]*Directive),
		Name:       ident,
	}
	for l.Peek() == '@' {
		d := parseDirective(l)
		fs.Directives[d.Name] = d
	}
	return fs
}

func parseArgument(l *lexer.Lexer) (string, Value) {
	name := l.ConsumeIdent()
	l.ConsumeToken(':')
	value := parseValue(l)
	return name, value
}

func parseValue(l *lexer.Lexer) Value {
	switch l.Peek() {
	case '$':
		l.ConsumeToken('$')
		return &Variable{Name: l.ConsumeIdent()}
	case scanner.Int:
		return &Literal{Value: l.ConsumeInt()}
	case scanner.Float:
		return &Literal{Value: l.ConsumeFloat()}
	case scanner.String:
		return &Literal{Value: l.ConsumeString()}
	case scanner.Ident:
		switch ident := l.ConsumeIdent(); ident {
		case "true":
			return &Literal{Value: true}
		case "false":
			return &Literal{Value: false}
		default:
			return &Literal{Value: ident}
		}
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
