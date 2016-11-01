package common

import (
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type InputMap struct {
	Fields     map[string]*InputValue
	FieldOrder []string
}

type InputValue struct {
	Name    string
	Type    Type
	Default interface{}
}

func ParseInputValue(l *lexer.Lexer) *InputValue {
	p := &InputValue{}
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	p.Type = ParseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = ParseValue(l, true)
	}
	return p
}

type Variable string

func ParseValue(l *lexer.Lexer, constOnly bool) interface{} {
	if !constOnly && l.Peek() == '$' {
		l.ConsumeToken('$')
		return Variable(l.ConsumeIdent())
	}

	switch l.Peek() {
	case scanner.Int:
		return l.ConsumeInt()
	case scanner.Float:
		return l.ConsumeFloat()
	case scanner.String:
		return l.ConsumeString()
	case scanner.Ident:
		switch ident := l.ConsumeIdent(); ident {
		case "true":
			return true
		case "false":
			return false
		default:
			return ident
		}
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
