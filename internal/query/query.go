package query

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Document struct {
	Operations map[string]*Operation
	Fragments  map[string]*Fragment
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

type Fragment struct {
	Name   string
	Type   string
	SelSet *SelectionSet
}

type SelectionSet struct {
	Selections []Selection
}

type Selection interface {
	isSelection()
}

type Field struct {
	Alias     string
	Name      string
	Arguments map[string]Value
	SelSet    *SelectionSet
}

type FragmentSpread struct {
	Name string
}

func (Field) isSelection()          {}
func (FragmentSpread) isSelection() {}

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

func Parse(queryString string) (res *Document, errRes error) {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(queryString))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(lexer.SyntaxError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	return parseDocument(lexer.New(sc)), nil
}

func parseDocument(l *lexer.Lexer) *Document {
	d := &Document{
		Operations: make(map[string]*Operation),
		Fragments:  make(map[string]*Fragment),
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

func parseFragment(l *lexer.Lexer) *Fragment {
	f := &Fragment{}
	f.Name = l.ConsumeIdent()
	l.ConsumeKeyword("on")
	f.Type = l.ConsumeIdent()
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
		return parseFragmentSpread(l)
	}
	return parseField(l)
}

func parseField(l *lexer.Lexer) *Field {
	f := &Field{
		Arguments: make(map[string]Value),
	}
	f.Alias = l.ConsumeIdent()
	f.Name = f.Alias
	if l.Peek() == ':' {
		l.ConsumeToken(':')
		f.Name = l.ConsumeIdent()
	}
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		if l.Peek() != ')' {
			name, value := parseArgument(l)
			f.Arguments[name] = value
			for l.Peek() != ')' {
				l.ConsumeToken(',')
				name, value := parseArgument(l)
				f.Arguments[name] = value
			}
		}
		l.ConsumeToken(')')
	}
	if l.Peek() == '{' {
		f.SelSet = parseSelectionSet(l)
	}
	return f
}

func parseFragmentSpread(l *lexer.Lexer) *FragmentSpread {
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	return &FragmentSpread{Name: l.ConsumeIdent()}
}

func parseArgument(l *lexer.Lexer) (string, Value) {
	name := l.ConsumeIdent()
	l.ConsumeToken(':')
	value := parseValue(l)
	return name, value
}

type ValueType int

const (
	Int ValueType = iota
	Float
	String
	Boolean
	Enum
)

func parseValue(l *lexer.Lexer) Value {
	switch l.Peek() {
	case '$':
		l.ConsumeToken('$')
		return &Variable{
			Name: l.ConsumeIdent(),
		}
	case scanner.String:
		return &Literal{
			Value: l.ConsumeString(),
		}
	case scanner.Ident:
		return &Literal{
			Value: l.ConsumeIdent(),
		}
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
