package extension

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/complexity"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
)

const errComplexityLimit = "COMPLEXITY_LIMIT_EXCEEDED"

// ComplexityLimit allows you to define a limit on query complexity
//
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
type ComplexityLimit struct {
	Func func(ctx context.Context, opCtx *graphql.OperationContext) int

	es   graphql.ExecutableSchema
	opts []complexity.Option
}

var _ interface {
	graphql.OperationContextMutator
	graphql.HandlerExtension
} = &ComplexityLimit{}

const complexityExtension = "ComplexityLimit"

type ComplexityStats struct {
	// The calculated complexity for this request
	Complexity int

	// The complexity limit for this request returned by the extension func
	ComplexityLimit int
}

// FixedComplexityLimit sets a complexity limit that does not change
func FixedComplexityLimit(limit int, opts ...complexity.Option) *ComplexityLimit {
	return &ComplexityLimit{
		Func: func(ctx context.Context, opCtx *graphql.OperationContext) int {
			return limit
		},
		opts: opts,
	}
}

func (c ComplexityLimit) ExtensionName() string {
	return complexityExtension
}

func (c *ComplexityLimit) Validate(schema graphql.ExecutableSchema) error {
	if c.Func == nil {
		return errors.New("ComplexityLimit func can not be nil")
	}
	c.es = schema
	return nil
}

func (c ComplexityLimit) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	op := opCtx.Doc.Operations.ForName(opCtx.OperationName)
	complexityCalcs := complexity.Calculate(ctx, c.es, op, opCtx.Variables, c.opts...)

	limit := c.Func(ctx, opCtx)

	opCtx.Stats.SetExtension(complexityExtension, &ComplexityStats{
		Complexity:      complexityCalcs,
		ComplexityLimit: limit,
	})

	if complexityCalcs > limit {
		err := gqlerror.Errorf(
			"operation has complexity %d, which exceeds the limit of %d",
			complexityCalcs,
			limit,
		)
		errcode.Set(err, errComplexityLimit)
		return err
	}

	return nil
}

func GetComplexityStats(ctx context.Context) *ComplexityStats {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx == nil {
		return nil
	}

	s, _ := opCtx.Stats.GetExtension(complexityExtension).(*ComplexityStats)
	return s
}
