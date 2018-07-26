package opentracing

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/vektah/gqlgen/graphql"
)

func ResolverMiddleware() graphql.FieldMiddleware {
	return func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		rctx := graphql.GetResolverContext(ctx)
		span, ctx := opentracing.StartSpanFromContext(ctx, rctx.Object+"_"+rctx.Field.Name,
			opentracing.Tag{Key: "resolver.object", Value: rctx.Object},
			opentracing.Tag{Key: "resolver.field", Value: rctx.Field.Name},
		)
		defer span.Finish()
		ext.SpanKind.Set(span, "server")
		ext.Component.Set(span, "gqlgen")

		res, err := next(ctx)

		if err != nil {
			ext.Error.Set(span, true)
			span.LogFields(
				log.String("event", "error"),
				log.String("message", err.Error()),
				log.String("error.kind", fmt.Sprintf("%T", err)),
			)
		}

		return res, err
	}
}

func RequestMiddleware() graphql.RequestMiddleware {
	return func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
		requestContext := graphql.GetRequestContext(ctx)
		span, ctx := opentracing.StartSpanFromContext(ctx, requestContext.RawQuery)
		defer span.Finish()
		ext.SpanKind.Set(span, "server")
		ext.Component.Set(span, "gqlgen")

		res := next(ctx)

		return res
	}
}
