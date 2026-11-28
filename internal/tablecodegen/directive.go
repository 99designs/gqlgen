package tablecodegen

import (
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
)

func locationDirectives(
	directives codegen.DirectiveList,
	location ...ast.DirectiveLocation,
) map[string]*codegen.Directive {
	mDirectives := make(map[string]*codegen.Directive)
	for name, d := range directives {
		if d.IsLocation(location...) {
			mDirectives[name] = d
		}
	}
	return mDirectives
}
