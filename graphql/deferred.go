package graphql

import (
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

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
	Errors gqlerror.List
}
