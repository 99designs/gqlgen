package common

import (
	"fmt"

	"github.com/neelance/graphql-go/internal/lexer"
)

type Type interface {
	Kind() string
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

func (*List) Kind() string     { return "LIST" }
func (*NonNull) Kind() string  { return "NON_NULL" }
func (*TypeName) Kind() string { panic("TypeName needs to be resolved to actual type") }

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

type Resolver func(name string) Type

func ResolveType(t Type, resolver Resolver) (Type, error) {
	switch t := t.(type) {
	case *List:
		ofType, err := ResolveType(t.OfType, resolver)
		if err != nil {
			return nil, err
		}
		return &List{OfType: ofType}, nil
	case *NonNull:
		ofType, err := ResolveType(t.OfType, resolver)
		if err != nil {
			return nil, err
		}
		return &NonNull{OfType: ofType}, nil
	case *TypeName:
		refT := resolver(t.Name)
		if refT == nil {
			return nil, fmt.Errorf("type %q not found", t.Name)
		}
		return refT, nil
	default:
		panic("unreachable")
	}
}
