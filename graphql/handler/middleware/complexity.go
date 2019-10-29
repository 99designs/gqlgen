package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

// ComplexityLimit sets a maximum query complexity that is allowed to be executed.
//
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
type ComplexityLimit int

var _ graphql.RequestContextMutator = ComplexityLimit(0)

func (c ComplexityLimit) MutateRequestContext(ctx context.Context, rc *graphql.RequestContext) *gqlerror.Error {
	rc.ComplexityLimit = int(c)
	return nil
}
