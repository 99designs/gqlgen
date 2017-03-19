package common

import (
	"github.com/neelance/graphql-go/internal/lexer"
)

func ParseDirectives(l *lexer.Lexer) map[string]ArgumentList {
	directives := make(map[string]ArgumentList)
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		name := l.ConsumeIdent()
		var args ArgumentList
		if l.Peek() == '(' {
			args = ParseArguments(l)
		}
		directives[name] = args
	}
	return directives
}
