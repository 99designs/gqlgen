package extension

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

// EnableIntrospection enables clients to reflect all of the types available on the graph.
type Introspection struct{}

var _ graphql.OperationContextMutator = Introspection{}

func (c Introspection) MutateOperationContext(ctx context.Context, rc *graphql.OperationContext) *gqlerror.Error {
	rc.DisableIntrospection = false
	return nil
}
