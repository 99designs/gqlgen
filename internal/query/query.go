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

type Field struct {
	Name string
	Sel  *SelectionSet
}

func parseField(l *lexer.Lexer) *Field {
	f := &Field{}
	f.Name = l.ConsumeIdent()
	if l.Peek() == '{' {
		f.Sel = parseSelectionSet(l)
	}
	return f
}
