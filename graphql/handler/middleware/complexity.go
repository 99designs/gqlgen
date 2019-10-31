package middleware

import (
	"context"

	"github.com/99designs/gqlgen/complexity"
	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

// ComplexityLimit allows you to define a limit on query complexity
//
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
type ComplexityLimit func(ctx context.Context, rc *graphql.RequestContext) int

var _ graphql.RequestContextMutator = ComplexityLimit(func(ctx context.Context, rc *graphql.RequestContext) int { return 0 })

func (c ComplexityLimit) MutateRequestContext(ctx context.Context, rc *graphql.RequestContext) *gqlerror.Error {
	es := graphql.GetServerContext(ctx)
	op := rc.Doc.Operations.ForName(rc.OperationName)
	complexity := complexity.Calculate(es, op, rc.Variables)

	limit := c(ctx, rc)

	if complexity > limit {
		return gqlerror.Errorf("operation has complexity %d, which exceeds the limit of %d", complexity, limit)
	}

	return nil
}
