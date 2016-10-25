package common

import (
	"text/scanner"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Value interface {
	Eval(vars map[string]interface{}) interface{}
}

type Variable struct {
	Name string
}

type Literal struct {
	Value interface{}
}

func (v *Variable) Eval(vars map[string]interface{}) interface{} {
	return vars[v.Name]
}

func (l *Literal) Eval(vars map[string]interface{}) interface{} {
	return l.Value
}

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
