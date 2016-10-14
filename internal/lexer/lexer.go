package lexer

import (
	"fmt"
	"strconv"
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
	if l.next == '#' {
		for {
			next := l.sc.Next()
			if next == '\n' || next == scanner.EOF {
				break
			}
		}
		l.Consume()
	}
}

func (l *Lexer) ConsumeIdent() string {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return text
}

func (l *Lexer) ConsumeString() string {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.String)
	value, err := strconv.Unquote(text)
	if err != nil {
		l.SyntaxError(err.Error())
	}
	return value
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %s", l.sc.TokenText(), scanner.TokenString(expected)))
	}
	l.Consume()
}

func (l *Lexer) SyntaxError(message string) {
	panic(SyntaxError(fmt.Sprintf("%s:%d: syntax error: %s", l.sc.Filename, l.sc.Line, message)))
}
