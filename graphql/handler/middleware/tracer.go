package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

// Tracer allows you to add a request/resolver tracer that will be called around the root request,
// calling resolver. This is useful for tracing
func Tracer(tracer graphql.Tracer) graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			rc := graphql.GetRequestContext(ctx)
			rc.AddTracer(tracer)
			rc.AddRequestMiddleware(func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
				ctx = tracer.StartOperationExecution(ctx)
				resp := next(ctx)
				tracer.EndOperationExecution(ctx)

				return resp
			})
			next(ctx, writer)
		}
	}
}
