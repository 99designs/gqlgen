package handler

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

// ComplexityLimit sets a maximum query complexity that is allowed to be executed.
//
// If a query is submitted that exceeds the limit, a 422 status code will be returned.
func ComplexityLimit(limit int) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, writer Writer) {
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
func ComplexityLimitFunc(f graphql.ComplexityLimitFunc) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, writer Writer) {
			graphql.GetRequestContext(ctx).ComplexityLimit = f(ctx)
			next(ctx, writer)
		}
	}
}
