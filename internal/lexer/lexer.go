package lexer

import (
	"fmt"
	"text/scanner"
)

type SyntaxError string

type Lexer struct {
	sc   *scanner.Scanner
	next rune
}

func New(sc *scanner.Scanner) *Lexer {
	l := &Lexer{sc: sc}
	l.Consume()
	return l
}

func (l *Lexer) Peek() rune {
	return l.next
}

func (l *Lexer) Consume() {
	l.next = l.sc.Scan()
}

func (l *Lexer) ConsumeIdent() string {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return text
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(scanner.TokenString(expected))
	}
	l.Consume()
}

func (l *Lexer) SyntaxError(expected string) {
	panic(SyntaxError(fmt.Sprintf("%s:%d: syntax error: unexpected %q, expecting %s", l.sc.Filename, l.sc.Line, l.sc.TokenText(), expected)))
}
