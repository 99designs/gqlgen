package graphql

import (
	"context"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Deferrable struct {
	Label string
}

type DeferredGroup struct {
	Path     ast.Path
	FieldSet *FieldSet
	Defers   map[string]*FieldSetView
	Context  context.Context
}

// NewDeferredGroup returns an empty group ready to collect the @defer'd fields
// of the object being resolved at ctx. The incremental payloads the group
// eventually emits are labelled with ctx's path and resolved under ctx, so it
// must be the same context the object's fields are collected and dispatched
// with.
func NewDeferredGroup(ctx context.Context) DeferredGroup {
	return DeferredGroup{
		Path:     GetPath(ctx),
		FieldSet: NewFieldSet(nil),
		Defers:   make(map[string]*FieldSetView),
		Context:  ctx,
	}
}

type DeferredResult struct {
	Path   ast.Path
	Label  string
	Result Marshaler
	Errors gqlerror.List
}
