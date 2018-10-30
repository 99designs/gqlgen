package gqlapollotracing

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

func RequestMiddleware() graphql.RequestMiddleware {
	return func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
		res := next(ctx)

		reqCtx := graphql.GetRequestContext(ctx)
		td := getTracingData(ctx)
		td.prepare()

		reqCtx.RegisterExtension("tracing", td)

		return res
	}
}
