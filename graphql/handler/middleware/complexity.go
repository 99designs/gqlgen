package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

// ComplexityLimit sets a maximum query complexity that is allowed to be executed.
//
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
func ComplexityLimit(limit int) graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			graphql.GetRequestContext(ctx).ComplexityLimit = limit
			next(ctx, writer)
		}
	}
}

// ComplexityLimitFunc allows you to define a function to dynamically set the maximum query complexity that is allowed
// to be executed. This is mostly just a wrapper to preserve the old interface, consider writing your own middleware
// instead.
//
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
func ComplexityLimitFunc(f graphql.ComplexityLimitFunc) graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			graphql.GetRequestContext(ctx).ComplexityLimit = f(ctx)
			next(ctx, writer)
		}
	}
}
