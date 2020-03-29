package extension

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Parallel()

	t.Run("Validate", func(t *testing.T) {
		ext := Cache{}
		require.NoError(t, ext.Validate(nil))
	})

	t.Run("InterceptResponse", func(t *testing.T) {
		t.Run("should inject CacheControl in context", func(t *testing.T) {
			ext := Cache{}

			ctx := context.Background()
			_ = ext.InterceptResponse(ctx, func(ctx context.Context) *graphql.Response {
				cc := graphql.CacheControl(ctx)
				require.NotNil(t, cc)
				return &graphql.Response{}
			})
		})

		t.Run("should not inject cacheControl extension", func(t *testing.T) {
			ext := Cache{}

			ctx := context.Background()
			resp := ext.InterceptResponse(ctx, func(ctx context.Context) *graphql.Response {
				return &graphql.Response{}
			})

			require.Nil(t, resp.Extensions["cacheControl"])
		})

		t.Run("should inject cacheControl extension", func(t *testing.T) {
			ext := Cache{}

			ctx := context.Background()
			resp := ext.InterceptResponse(ctx, func(ctx context.Context) *graphql.Response {
				cc := graphql.CacheControl(ctx)
				cc.AddHint(graphql.Hint{
					MaxAge: 10,
					Scope:  graphql.CacheScopePrivate,
				})
				return &graphql.Response{}
			})

			require.NotNil(t, resp.Extensions["cacheControl"])
		})

		t.Run("should not override extensions", func(t *testing.T) {
			ext := Cache{}

			ctx := context.Background()
			resp := ext.InterceptResponse(ctx, func(ctx context.Context) *graphql.Response {
				return &graphql.Response{
					Extensions: map[string]interface{}{
						"foo": "bar",
					},
				}
			})

			require.NotNil(t, resp.Extensions["foo"])
			require.Nil(t, resp.Extensions["cacheControl"])
		})
	})

}
