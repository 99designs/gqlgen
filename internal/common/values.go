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
	Desc    string
}

func ParseInputValue(l *lexer.Lexer) *InputValue {
	p := &InputValue{}
	p.Desc = l.DescComment()
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
	switch l.Peek() {
	case '$':
		if constOnly {
			l.SyntaxError("variable not allowed")
			panic("unreachable")
		}
		l.ConsumeToken('$')
		return Variable(l.ConsumeIdent())
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
		case "null":
			return nil
		default:
			return ident
		}
	case '[':
		l.ConsumeToken('[')
		var list []interface{}
		for l.Peek() != ']' {
			list = append(list, ParseValue(l, constOnly))
		}
		l.ConsumeToken(']')
		return list
	case '{':
		l.ConsumeToken('{')
		obj := make(map[string]interface{})
		for l.Peek() != '}' {
			name := l.ConsumeIdent()
			l.ConsumeToken(':')
			obj[name] = ParseValue(l, constOnly)
		}
		l.ConsumeToken('}')
		return obj
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
