package common

import (
	"strconv"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/lexer"
)

type InputValue struct {
	Name    string
	Type    Type
	Default *ValueWithLoc
	Desc    string
}

type InputValueList []*InputValue

func (l InputValueList) Get(name string) *InputValue {
	for _, v := range l {
		if v.Name == name {
			return v
		}
	}
	return nil
}

type ValueWithLoc struct {
	Value interface{}
	Loc   *errors.Location
}

func ParseInputValue(l *lexer.Lexer) *InputValue {
	p := &InputValue{}
	p.Desc = l.DescComment()
	p.Name = l.ConsumeIdent()
	l.ConsumeToken(':')
	p.Type = ParseType(l)
	if l.Peek() == '=' {
		l.ConsumeToken('=')
		v := ParseValue(l, true)
		p.Default = &v
	}
	return p
}

type Argument struct {
	Name  string
	Value ValueWithLoc
}

type ArgumentList []Argument

func (l ArgumentList) Get(name string) ValueWithLoc {
	for _, arg := range l {
		if arg.Name == name {
			return arg.Value
		}
	}
	return ValueWithLoc{}
}

func ParseArguments(l *lexer.Lexer) ArgumentList {
	var args ArgumentList
	l.ConsumeToken('(')
	for l.Peek() != ')' {
		name := l.ConsumeIdent()
		l.ConsumeToken(':')
		value := ParseValue(l, false)
		args = append(args, Argument{Name: name, Value: value})
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
		return l.ConsumeVariable()
	case scanner.Int, scanner.Float, scanner.String, scanner.Ident:
		return l.ConsumeLiteral()
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

func UnmarshalLiteral(lit *lexer.Literal) interface{} {
	switch lit.Type {
	case scanner.Int, scanner.Float:
		value, err := strconv.ParseFloat(lit.Text, 64)
		if err != nil {
			panic(err)
		}
		return value

	case scanner.String:
		value, err := strconv.Unquote(lit.Text)
		if err != nil {
			panic(err)
		}
		return value

	case scanner.Ident:
		switch lit.Text {
		case "true":
			return true
		case "false":
			return false
		default:
			return lit.Text
		}

	default:
		panic("invalid literal")
	}
}
