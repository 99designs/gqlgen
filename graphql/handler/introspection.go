package handler

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

// Introspection enables clients to reflect all of the types available on the graph.
func Introspection() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, writer Writer) {
			graphql.GetRequestContext(ctx).DisableIntrospection = false
			next(ctx, writer)
		}
	}
}
