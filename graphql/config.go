package graphql

import "github.com/vektah/gqlparser/v2/ast"

// Config holds dependencies for constructing an executable schema.
// R, D, C are generated types from the target schema package.
type Config[R any, D any, C any] struct {
	Schema     *ast.Schema
	Resolvers  R
	Directives D
	Complexity C
}
