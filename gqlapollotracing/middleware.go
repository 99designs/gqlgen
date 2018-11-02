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

		err := reqCtx.RegisterExtension("tracing", td)
		if err != nil {
			reqCtx.Error(ctx, err)
		}

		return res
	}
}
