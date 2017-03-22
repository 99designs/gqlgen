package common

import (
	"github.com/neelance/graphql-go/internal/lexer"
)

type Directive struct {
	Name lexer.Ident
	Args ArgumentList
}

func ParseDirectives(l *lexer.Lexer) map[string]*Directive {
	directives := make(map[string]*Directive)
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		d := &Directive{}
		d.Name = l.ConsumeIdentWithLoc()
		d.Name.Loc.Column--
		if l.Peek() == '(' {
			d.Args = ParseArguments(l)
		}
		directives[d.Name.Name] = d
	}
	return directives
}
