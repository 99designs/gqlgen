package transport

import (
	"github.com/99designs/gqlgen/graphql/handler/cache"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestWriteCacheControl(t *testing.T) {
	t.Run("should write cache-control header when it has cacheControl in Response.Extensions", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := &graphql.Response{
			Extensions: map[string]interface{}{
				"cacheControl": &cache.CacheControlExtension{
					Version: 1,
					Hints: []cache.Hint{
						{MaxAge: time.Minute.Seconds(), Scope: cache.ScopePublic},
					},
				},
			},
		}
		writeCacheControl(w, r)

		require.Equal(t, w.Header().Get("Cache-Control"), "max-age: 60 public")
	})

	t.Run("should do nothing when it has no cache cacheControl in Response.Extensions", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := &graphql.Response{}
		writeCacheControl(w, r)

		require.Empty(t, w.Header().Get("Cache-Control"))
	})

	t.Run("should do nothing when Response has errors", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := &graphql.Response{
			Errors: []*gqlerror.Error{
				{},
			},
			Extensions: map[string]interface{}{
				"cacheControl": &cache.CacheControlExtension{
					Version: 1,
					Hints: []cache.Hint{
						{MaxAge: time.Minute.Seconds(), Scope: cache.ScopePublic},
					},
				},
			},
		}
		writeCacheControl(w, r)

		require.Empty(t, w.Header().Get("Cache-Control"))
	})
}
