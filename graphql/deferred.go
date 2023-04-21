package graphql

import "github.com/vektah/gqlparser/v2/ast"

type DeferredResult struct {
	Path   ast.Path
	Result Marshaler
}
