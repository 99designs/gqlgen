package extension

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/vektah/gqlparser/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/mitchellh/mapstructure"
)

const errPersistedQueryNotFound = "PersistedQueryNotFound"

// AutomaticPersistedQuery saves client upload by optimistically sending only the hashes of queries, if the server
// does not yet know what the query is for the hash it will respond telling the client to send the query along with the
// hash in the next request.
// see https://github.com/apollographql/apollo-link-persisted-queries
type AutomaticPersistedQuery struct {
	Cache graphql.Cache
}

var _ graphql.OperationParameterMutator = AutomaticPersistedQuery{}

func (a AutomaticPersistedQuery) ExtensionName() string {
	return "AutomaticPersistedQuery"
}

func (a AutomaticPersistedQuery) Validate(schema graphql.ExecutableSchema) error {
	if a.Cache == nil {
		return fmt.Errorf("AutomaticPersistedQuery.Cache can not be nil")
	}
	return nil
}

func (a AutomaticPersistedQuery) MutateOperationParameters(ctx context.Context, rawParams *graphql.RawParams) *gqlerror.Error {
	if rawParams.Extensions["persistedQuery"] == nil {
		return nil
	}

	var extension struct {
		Sha256  string `json:"sha256Hash"`
		Version int64  `json:"version"`
	}

	if err := mapstructure.Decode(rawParams.Extensions["persistedQuery"], &extension); err != nil {
		return gqlerror.Errorf("invalid APQ extension data")
	}

	if extension.Version != 1 {
		return gqlerror.Errorf("unsupported APQ version")
	}

	if rawParams.Query == "" {
		// client sent optimistic query hash without query string, get it from the cache
		query, ok := a.Cache.Get(extension.Sha256)
		if !ok {
			return gqlerror.Errorf(errPersistedQueryNotFound)
		}
		rawParams.Query = query.(string)
	} else {
		// client sent optimistic query hash with query string, verify and store it
		if computeQueryHash(rawParams.Query) != extension.Sha256 {
			return gqlerror.Errorf("provided APQ hash does not match query")
		}
		a.Cache.Add(extension.Sha256, rawParams.Query)
	}

	return nil
}

func computeQueryHash(query string) string {
	b := sha256.Sum256([]byte(query))
	return hex.EncodeToString(b[:])
}
