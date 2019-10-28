package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/99designs/gqlgen/graphql"
	"github.com/mitchellh/mapstructure"
)

const (
	errPersistedQueryNotSupported = "PersistedQueryNotSupported"
	errPersistedQueryNotFound     = "PersistedQueryNotFound"
)

// AutomaticPersistedQuery saves client upload by optimistically sending only the hashes of queries, if the server
// does not yet know what the query is for the hash it will respond telling the client to send the query along with the
// hash in the next request.
// see https://github.com/apollographql/apollo-link-persisted-queries
func AutomaticPersistedQuery(cache graphql.Cache) graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			rc := graphql.GetRequestContext(ctx)

			if rc.Extensions["persistedQuery"] == nil {
				next(ctx, writer)
				return
			}

			var extension struct {
				Sha256  string `json:"sha256Hash"`
				Version int64  `json:"version"`
			}

			if err := mapstructure.Decode(rc.Extensions["persistedQuery"], &extension); err != nil {
				writer.Error("Invalid APQ extension data")
				return
			}

			if extension.Version != 1 {
				writer.Error("Unsupported APQ version")
				return
			}

			if rc.RawQuery == "" {
				// client sent optimistic query hash without query string, get it from the cache
				query, ok := cache.Get(extension.Sha256)
				if !ok {
					writer.Error(errPersistedQueryNotFound)
					return
				}
				rc.RawQuery = query.(string)
			} else {
				// client sent optimistic query hash with query string, verify and store it
				if computeQueryHash(rc.RawQuery) != extension.Sha256 {
					writer.Error("Provided APQ hash does not match query")
					return
				}
				cache.Add(extension.Sha256, rc.RawQuery)
			}
			next(ctx, writer)
		}
	}
}

func computeQueryHash(query string) string {
	b := sha256.Sum256([]byte(query))
	return hex.EncodeToString(b[:])
}
