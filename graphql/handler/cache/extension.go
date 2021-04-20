package cache

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type Extension struct{}

var _ graphql.HandlerExtension = Extension{}
var _ graphql.ResponseInterceptor = Extension{}

func (c Extension) ExtensionName() string {
	return "cache"
}

func (c Extension) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

func (c Extension) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	ctx = WithCacheControlExtension(ctx)

	result := next(ctx)

	if result != nil {
		cache := CacheControl(ctx)

		if len(cache.Hints) > 0 {
			if result.Extensions == nil {
				result.Extensions = make(map[string]interface{})
			}
			result.Extensions["cacheControl"] = cache
		}
	}

	return result
}
