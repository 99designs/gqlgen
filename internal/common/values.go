package common

import (
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
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

type ValueWithLoc struct {
	Value interface{}
	Loc   *errors.Location
}

type Variable string

type EnumValue string

func ParseInputValue(l *lexer.Lexer) *InputValue {
	p := &InputValue{}
	p.Desc = l.DescComment()
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	p.Type = ParseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		p.Default = parseValue(l, true)
	}
	return p
}

func ParseArguments(l *lexer.Lexer) map[string]ValueWithLoc {
	args := make(map[string]ValueWithLoc)
	l.ConsumeToken('(')
	for l.Peek() != ')' {
		name := l.ConsumeIdent()
		l.ConsumeToken(':')
		value := ParseValue(l, false)
		args[name] = value
	}
	l.ConsumeToken(')')
	return args
}

func ParseValue(l *lexer.Lexer, constOnly bool) ValueWithLoc {
	loc := l.Location()
	value := parseValue(l, constOnly)
	return ValueWithLoc{
		Value: value,
		Loc:   loc,
	}
}

func parseValue(l *lexer.Lexer, constOnly bool) interface{} {
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
		return parseIdent(l)
	case '[':
		l.ConsumeToken('[')
		var list []interface{}
		for l.Peek() != ']' {
			list = append(list, parseValue(l, constOnly))
		}
		l.ConsumeToken(']')
		return list
	case '{':
		l.ConsumeToken('{')
		obj := make(map[string]interface{})
		for l.Peek() != '}' {
			name := l.ConsumeIdent()
			l.ConsumeToken(':')
			obj[name] = parseValue(l, constOnly)
		}
		l.ConsumeToken('}')
		return obj
	default:
		l.SyntaxError("invalid value")
		panic("unreachable")
	}
}

func parseIdent(l *lexer.Lexer) interface{} {
	switch ident := l.ConsumeIdent(); ident {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	default:
		return EnumValue(ident)
	}
}
