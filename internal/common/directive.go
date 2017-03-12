package common

import (
	"github.com/neelance/graphql-go/internal/lexer"
)

type Directive struct {
	Name string
	Args map[string]interface{}
}

func ParseDirectives(l *lexer.Lexer) map[string]*Directive {
	directives := make(map[string]*Directive)
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		name := l.ConsumeIdent()
		var args map[string]interface{}
		if l.Peek() == '(' {
			args = ParseArguments(l)
		}
		directives[name] = &Directive{Name: name, Args: args}
	}
	return directives
}
