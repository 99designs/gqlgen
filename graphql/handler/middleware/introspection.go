package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

// Introspection enables clients to reflect all of the types available on the graph.
func Introspection() graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			graphql.GetRequestContext(ctx).DisableIntrospection = false
			next(ctx, writer)
		}
	}
}
