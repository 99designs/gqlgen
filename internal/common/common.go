package common

import "github.com/neelance/graphql-go/internal/lexer"

type Type interface {
	IsType()
}

type List struct {
	OfType Type
}

type NonNull struct {
	OfType Type
}

type TypeName struct {
	Name string
}

func (*List) IsType()     {}
func (*NonNull) IsType()  {}
func (*TypeName) IsType() {}

func ParseType(l *lexer.Lexer) Type {
	t := parseNullType(l)
	if l.Peek() == '!' {
		l.ConsumeToken('!')
		return &NonNull{OfType: t}
	}
	return t
}

func parseNullType(l *lexer.Lexer) Type {
	if l.Peek() == '[' {
		l.ConsumeToken('[')
		ofType := ParseType(l)
		l.ConsumeToken(']')
		return &List{OfType: ofType}
	}

	return &TypeName{Name: l.ConsumeIdent()}
}
