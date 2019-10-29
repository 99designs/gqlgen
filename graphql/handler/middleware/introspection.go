package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

// EnableIntrospection enables clients to reflect all of the types available on the graph.
type Introspection struct{}

var _ graphql.RequestContextMutator = Introspection{}

func (c Introspection) MutateRequestContext(ctx context.Context, rc *graphql.RequestContext) *gqlerror.Error {
	rc.DisableIntrospection = false
	return nil
}
