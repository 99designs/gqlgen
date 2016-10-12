package graphql

import (
	"fmt"
	"text/scanner"
)

type lexer struct {
	sc   *scanner.Scanner
	next rune
}

func newLexer(sc *scanner.Scanner) *lexer {
	l := &lexer{sc: sc}
	l.consume()
	return l
}

func (l *lexer) peek() rune {
	return l.next
}

func (l *lexer) consume() {
	l.next = l.sc.Scan()
}

func (l *lexer) consumeIdent() string {
	text := l.sc.TokenText()
	l.consumeToken(scanner.Ident)
	return text
}

func (l *lexer) consumeToken(expected rune) {
	if l.next != expected {
		l.syntaxError(scanner.TokenString(expected))
	}
	l.consume()
}

func (l *lexer) syntaxError(expected string) {
	panic(parseError(fmt.Sprintf("%s:%d: syntax error: unexpected %q, expecting %s", l.sc.Filename, l.sc.Line, l.sc.TokenText(), expected)))
}
