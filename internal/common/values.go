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
	Default Value
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

type Value interface {
	isValue()
}

type Variable struct {
	Name string
}

type Literal struct {
	Value interface{}
}

func (*Variable) isValue() {}
func (*Literal) isValue()  {}

func ParseValue(l *lexer.Lexer, constOnly bool) Value {
	if !constOnly && l.Peek() == '$' {
		l.ConsumeToken('$')
		return &Variable{Name: l.ConsumeIdent()}
	}

	switch l.Peek() {
	case scanner.Int:
		return &Literal{Value: l.ConsumeInt()}
	case scanner.Float:
		return &Literal{Value: l.ConsumeFloat()}
	case scanner.String:
		return &Literal{Value: l.ConsumeString()}
	case scanner.Ident:
		switch ident := l.ConsumeIdent(); ident {
		case "true":
			return &Literal{Value: true}
		case "false":
			return &Literal{Value: false}
		default:
			return &Literal{Value: ident}
		}
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}
