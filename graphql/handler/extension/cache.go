package extension

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type Cache struct{}

var _ graphql.HandlerExtension = Cache{}
var _ graphql.ResponseInterceptor = Cache{}

func (c Cache) ExtensionName() string {
	return "cache"
}

func (c Cache) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

func (c Cache) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	cache := &graphql.CacheControlExtension{Version: 1}
	ctx = context.WithValue(ctx, graphql.CacheKey, cache)

	result := next(ctx)

	if result != nil && len(cache.Hints) > 0 {
		if result.Extensions == nil {
			result.Extensions = make(map[string]interface{})
		}
		result.Extensions["cacheControl"] = cache
	}

	return result
}
