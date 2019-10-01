package handler

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
)

func TestAPQ(t *testing.T) {
	const query = "{ me { name } }"
	const hash = "b8d9506e34c83b0e53c2aa463624fcea354713bc38f95276e6f0bd893ffb5b88"

	t.Run("with query and no hash", func(t *testing.T) {
		rc := testMiddleware(AutomaticPersistedQuery(MapCache{}), graphql.RequestContext{
			RawQuery: "original query",
		})

		require.True(t, rc.InvokedNext)
		require.Equal(t, "original query", rc.ResultContext.RawQuery)
	})

	t.Run("with hash miss and no query", func(t *testing.T) {
		rc := testMiddleware(AutomaticPersistedQuery(MapCache{}), graphql.RequestContext{
			RawQuery: "",
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		})

		require.False(t, rc.InvokedNext)
		require.Equal(t, "PersistedQueryNotFound", rc.Response.Errors[0].Message)
	})

	t.Run("with hash miss and query", func(t *testing.T) {
		cache := MapCache{}
		rc := testMiddleware(AutomaticPersistedQuery(cache), graphql.RequestContext{
			RawQuery: query,
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		})

		require.True(t, rc.InvokedNext, rc.Response.Errors)
		require.Equal(t, "{ me { name } }", rc.ResultContext.RawQuery)
		require.Equal(t, "{ me { name } }", cache[hash])
	})

	t.Run("with hash miss and query", func(t *testing.T) {
		cache := MapCache{}
		rc := testMiddleware(AutomaticPersistedQuery(cache), graphql.RequestContext{
			RawQuery: query,
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		})

		require.True(t, rc.InvokedNext, rc.Response.Errors)
		require.Equal(t, "{ me { name } }", rc.ResultContext.RawQuery)
		require.Equal(t, "{ me { name } }", cache[hash])
	})

	t.Run("with hash hit and no query", func(t *testing.T) {
		cache := MapCache{
			hash: query,
		}
		rc := testMiddleware(AutomaticPersistedQuery(cache), graphql.RequestContext{
			RawQuery: "",
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		})

		require.True(t, rc.InvokedNext, rc.Response.Errors)
		require.Equal(t, "{ me { name } }", rc.ResultContext.RawQuery)
	})

	t.Run("with malformed extension payload", func(t *testing.T) {
		rc := testMiddleware(AutomaticPersistedQuery(MapCache{}), graphql.RequestContext{
			Extensions: map[string]interface{}{
				"persistedQuery": "asdf",
			},
		})

		require.False(t, rc.InvokedNext)
		require.Equal(t, "Invalid APQ extension data", rc.Response.Errors[0].Message)
	})

	t.Run("with invalid extension version", func(t *testing.T) {
		rc := testMiddleware(AutomaticPersistedQuery(MapCache{}), graphql.RequestContext{
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"version": 2,
				},
			},
		})

		require.False(t, rc.InvokedNext)
		require.Equal(t, "Unsupported APQ version", rc.Response.Errors[0].Message)
	})

	t.Run("with hash mismatch", func(t *testing.T) {
		rc := testMiddleware(AutomaticPersistedQuery(MapCache{}), graphql.RequestContext{
			RawQuery: query,
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  "badhash",
					"version": 1,
				},
			},
		})

		require.False(t, rc.InvokedNext)
		require.Equal(t, "Provided APQ hash does not match query", rc.Response.Errors[0].Message)
	})
}
