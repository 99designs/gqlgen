package query

import (
	"errors"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type SelectionSet struct {
	Selections []*Field
}

type Field struct {
	Alias     string
	Name      string
	Arguments map[string]*Value
	Sel       *SelectionSet
}

type Value struct {
	Value interface{}
}

func Parse(queryString string) (res *SelectionSet, errRes error) {
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

	return parseSelectionSet(lexer.New(sc)), nil
}

func parseSelectionSet(l *lexer.Lexer) *SelectionSet {
	sel := &SelectionSet{}
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		sel.Selections = append(sel.Selections, parseField(l))
	}
	l.ConsumeToken('}')
	return sel
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
		f.Sel = parseSelectionSet(l)
	}
	return f
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
