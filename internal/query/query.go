package query

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Query struct {
	Root      *SelectionSet
	Fragments map[string]*Fragment
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
	Arguments map[string]*Value
	SelSet    *SelectionSet
}

type FragmentSpread struct {
	Name string
}

func (Field) isSelection()          {}
func (FragmentSpread) isSelection() {}

type Value struct {
	Value interface{}
}

func Parse(queryString string) (res *Query, errRes error) {
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

	return parseQuery(lexer.New(sc)), nil
}

func parseQuery(l *lexer.Lexer) *Query {
	q := &Query{
		Fragments: make(map[string]*Fragment),
	}
	for l.Peek() != scanner.EOF {
		if l.Peek() == '{' {
			q.Root = parseSelectionSet(l)
			continue
		}

		switch x := l.ConsumeIdent(); x {
		case "fragment":
			f := parseFragment(l)
			q.Fragments[f.Name] = f

		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "fragment"`, x))
		}
	}
	return q
}

func parseFragment(l *lexer.Lexer) *Fragment {
	f := &Fragment{}
	f.Name = l.ConsumeIdent()
	l.ConsumeKeyword("on")
	f.Type = l.ConsumeIdent()
	f.SelSet = parseSelectionSet(l)
	return f
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
		Arguments: make(map[string]*Value),
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

func parseArgument(l *lexer.Lexer) (string, *Value) {
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

func parseValue(l *lexer.Lexer) *Value {
	switch l.Peek() {
	case scanner.String:
		return &Value{
			Value: l.ConsumeString(),
		}
	case scanner.Ident:
		return &Value{
			Value: l.ConsumeIdent(),
		}
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
