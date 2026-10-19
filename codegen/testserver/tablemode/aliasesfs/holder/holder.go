// Package holder holds the object whose field takes BInput.
package holder

import (
	bmodel "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliases/b/model"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
)

type Holder struct{}

func (Holder) B(in *bmodel.Input) *string { return new(resolvers.Describe(in)) }
