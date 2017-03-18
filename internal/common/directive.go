package common

import (
	"github.com/neelance/graphql-go/internal/lexer"
)

type DirectiveArgs map[string]ValueWithLoc

func ParseDirectives(l *lexer.Lexer) map[string]DirectiveArgs {
	directives := make(map[string]DirectiveArgs)
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		name := l.ConsumeIdent()
		args := make(DirectiveArgs)
		if l.Peek() == '(' {
			args = ParseArguments(l)
		}
		directives[name] = args
	}
	return directives
}
