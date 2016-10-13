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
	Name      string
	Arguments map[string]*Value
	Sel       *SelectionSet
}

type Value struct {
	Value interface{}
}

func Parse(queryString string) (res *SelectionSet, errRes error) {
	sc := &scanner.Scanner{}
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
	f.Name = l.ConsumeIdent()
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

func parseValue(l *lexer.Lexer) *Value {
	value := l.ConsumeString()
	return &Value{Value: value}
}
