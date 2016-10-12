package query

import "github.com/neelance/graphql-go/internal/lexer"

type SelectionSet struct {
	Selections []*Field
}

func Parse(l *lexer.Lexer) *SelectionSet {
	return parseSelectionSet(l)
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
