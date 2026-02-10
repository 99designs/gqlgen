package federation

import (
	"github.com/vektah/gqlparser/v2/ast"
)

// federationDirectiveNames lists directives that are internal to Apollo
// Federation or gqlgen and must NOT be applied as user-facing runtime
// middleware when resolving entities via the _entities query.
var federationDirectiveNames = map[string]bool{
	// Federation v1
	"key":      true,
	"requires": true,
	"provides": true,
	"external": true,
	"extends":  true,

	// Federation v2
	"tag":              true,
	"inaccessible":     true,
	"shareable":        true,
	"interfaceObject":  true,
	"override":         true,
	"composeDirective": true,
	"link":             true,

	// Federation v2.5+
	"authenticated":  true,
	"requiresScopes": true,
	"policy":         true,

	// gqlgen-specific
	"entityResolver": true,
	"goModel":        true,
	"goField":        true,
	"goTag":          true,
}

// HasObjectDirectives reports whether this entity's type definition carries
// any non-federation user-defined directives that should be applied at
// runtime when the entity is resolved through the _entities query.
func (e *Entity) HasObjectDirectives() bool {
	return len(e.ImplDirectives) > 0
}

// NonFederationDirectives returns the AST directives applied to this entity,
// excluding federation-internal and gqlgen-internal directives.
func (e *Entity) NonFederationDirectives() ast.DirectiveList {
	if e.Def == nil {
		return nil
	}
	var result ast.DirectiveList
	for _, d := range e.Def.Directives {
		if !federationDirectiveNames[d.Name] {
			result = append(result, d)
		}
	}
	return result
}
