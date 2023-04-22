package graphql

import "github.com/vektah/gqlparser/v2/ast"

type Deferrable struct {
	Label string
}

type DeferredGroup struct {
	Path     ast.Path
	Label    string
	FieldSet *FieldSet
}

type DeferredResult struct {
	Path   ast.Path
	Label  string
	Result Marshaler
}
