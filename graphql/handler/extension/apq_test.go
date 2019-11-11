package extension

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
)

func TestAPQ(t *testing.T) {
	const query = "{ me { name } }"
	const hash = "b8d9506e34c83b0e53c2aa463624fcea354713bc38f95276e6f0bd893ffb5b88"

	t.Run("with query and no hash", func(t *testing.T) {
		params := &graphql.RawParams{
			Query: "original query",
		}
		err := AutomaticPersistedQuery{graphql.MapCache{}}.MutateOperationParameters(context.Background(), params)
		require.Nil(t, err)

		require.Equal(t, "original query", params.Query)
	})

	t.Run("with hash miss and no query", func(t *testing.T) {
		params := &graphql.RawParams{
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		}

		err := AutomaticPersistedQuery{graphql.MapCache{}}.MutateOperationParameters(context.Background(), params)
		require.Equal(t, err.Message, "PersistedQueryNotFound")
	})

	t.Run("with hash miss and query", func(t *testing.T) {
		params := &graphql.RawParams{
			Query: query,
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		}
		cache := graphql.MapCache{}
		err := AutomaticPersistedQuery{cache}.MutateOperationParameters(context.Background(), params)
		require.Nil(t, err)

		require.Equal(t, "{ me { name } }", params.Query)
		require.Equal(t, "{ me { name } }", cache[hash])
	})

	t.Run("with hash miss and query", func(t *testing.T) {
		params := &graphql.RawParams{
			Query: query,
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		}
		cache := graphql.MapCache{}
		err := AutomaticPersistedQuery{cache}.MutateOperationParameters(context.Background(), params)
		require.Nil(t, err)

		require.Equal(t, "{ me { name } }", params.Query)
		require.Equal(t, "{ me { name } }", cache[hash])
	})

	t.Run("with hash hit and no query", func(t *testing.T) {
		params := &graphql.RawParams{
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  hash,
					"version": 1,
				},
			},
		}
		cache := graphql.MapCache{
			hash: query,
		}
		err := AutomaticPersistedQuery{cache}.MutateOperationParameters(context.Background(), params)
		require.Nil(t, err)

		require.Equal(t, "{ me { name } }", params.Query)
	})

	t.Run("with malformed extension payload", func(t *testing.T) {
		params := &graphql.RawParams{
			Extensions: map[string]interface{}{
				"persistedQuery": "asdf",
			},
		}

		err := AutomaticPersistedQuery{graphql.MapCache{}}.MutateOperationParameters(context.Background(), params)
		require.Equal(t, err.Message, "invalid APQ extension data")
	})

	t.Run("with invalid extension version", func(t *testing.T) {
		params := &graphql.RawParams{
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"version": 2,
				},
			},
		}
		err := AutomaticPersistedQuery{graphql.MapCache{}}.MutateOperationParameters(context.Background(), params)
		require.Equal(t, err.Message, "unsupported APQ version")
	})

	t.Run("with hash mismatch", func(t *testing.T) {
		params := &graphql.RawParams{
			Query: query,
			Extensions: map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"sha256":  "badhash",
					"version": 1,
				},
			},
		}

		err := AutomaticPersistedQuery{graphql.MapCache{}}.MutateOperationParameters(context.Background(), params)
		require.Equal(t, err.Message, "provided APQ hash does not match query")
	})
}
