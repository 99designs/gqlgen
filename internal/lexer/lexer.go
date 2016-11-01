package lexer

import (
	"fmt"
	"strconv"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
)

type syntaxError string

type Lexer struct {
	sc   *scanner.Scanner
	next rune
}

func New(sc *scanner.Scanner) *Lexer {
	l := &Lexer{sc: sc}
	l.Consume()
	return l
}

func (l *Lexer) CatchSyntaxError(f func()) (errRes *errors.QueryError) {
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(syntaxError); ok {
				errRes = errors.ErrorfWithLoc(l.sc.Line, l.sc.Column, "syntax error: %s", err)
				return
			}
			panic(err)
		}
	}()

	f()
	return
}

func (l *Lexer) Peek() rune {
	return l.next
}

func (l *Lexer) Consume() {
	l.next = l.sc.Scan()
	if l.next == ',' {
		l.Consume()
		return
	}
	if l.next == '#' {
		for {
			next := l.sc.Next()
			if next == '\n' || next == scanner.EOF {
				break
			}
		}
		l.Consume()
		return
	}
}

func (l *Lexer) ConsumeIdent() string {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.Ident)
	return text
}

func (l *Lexer) ConsumeKeyword(keyword string) {
	if l.next != scanner.Ident || l.sc.TokenText() != keyword {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %q", l.sc.TokenText(), keyword))
	}
	l.Consume()
}

func (l *Lexer) ConsumeInt() int {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.Int)
	value, _ := strconv.Atoi(text)
	return value
}

func (l *Lexer) ConsumeFloat() float64 {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.Int)
	value, _ := strconv.ParseFloat(text, 64)
	return value
}

func (l *Lexer) ConsumeString() string {
	text := l.sc.TokenText()
	l.ConsumeToken(scanner.String)
	value, _ := strconv.Unquote(text)
	return value
}

func (l *Lexer) ConsumeToken(expected rune) {
	if l.next != expected {
		l.SyntaxError(fmt.Sprintf("unexpected %q, expecting %s", l.sc.TokenText(), scanner.TokenString(expected)))
	}
	l.Consume()
}

func (l *Lexer) SyntaxError(message string) {
	panic(syntaxError(message))
}
