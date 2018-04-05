package opentracing

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/vektah/gqlgen/graphql"
)

func Middleware() graphql.ResolverMiddleware {
	return func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		rctx := graphql.GetResolverContext(ctx)

		fmt.Println("SPAN", rctx.Object+"_"+rctx.Field.Name)
		span, ctx := opentracing.StartSpanFromContext(ctx, rctx.Object+"_"+rctx.Field.Name)
		defer span.Finish()

		res, err := next(ctx)

		span.LogFields(
			log.String("object", rctx.Object),
			log.String("name", rctx.Field.Name),
			log.String("alias", rctx.Field.Alias),
		)

		if err != nil {
			ext.Error.Set(span, true)
			span.LogFields(log.String("error", err.Error()))
		}

		return res, err
	}
}
