package graphql

import (
	"context"
	"time"

	"github.com/vektah/gqlparser/v2/ast"
)

// Cache is a shared store for APQ and query AST caching
type Cache interface {
	// Get looks up a key's value from the cache.
	Get(ctx context.Context, key string) (value interface{}, ok bool)

	// Add adds a value to the cache.
	Add(ctx context.Context, key string, value interface{})
}

// MapCache is the simplest implementation of a cache, because it can not evict it should only be used in tests
type MapCache map[string]interface{}

// Get looks up a key's value from the cache.
func (m MapCache) Get(ctx context.Context, key string) (value interface{}, ok bool) {
	v, ok := m[key]
	return v, ok
}

// Add adds a value to the cache.
func (m MapCache) Add(ctx context.Context, key string, value interface{}) { m[key] = value }

type NoCache struct{}

func (n NoCache) Get(ctx context.Context, key string) (value interface{}, ok bool) { return nil, false }
func (n NoCache) Add(ctx context.Context, key string, value interface{})           {}

type CacheScope string

const (
	CacheScopePublic  = CacheScope("PUBLIC")
	CacheScopePrivate = CacheScope("PRIVATE")
)

type Hint struct {
	Path   ast.Path   `json:"path"`
	MaxAge float64    `json:"maxAge"`
	Scope  CacheScope `json:"scope"`
}

type OverallCachePolicy struct {
	MaxAge float64
	Scope  CacheScope
}

type CacheControl struct {
	Version int    `json:"version"`
	Hints   []Hint `json:"hints"`
}

func (cache *CacheControl) AddHint(h Hint) {
	cache.Hints = append(cache.Hints, h)
}

// OverallPolicy return a calculated cache policy
// TODO should implement the spec. ref: https://www.apollographql.com/docs/apollo-server/performance/caching/#adding-cache-hints-statically-in-your-schema
func (cache CacheControl) OverallPolicy() OverallCachePolicy {
	var scope = CacheScopePublic
	var maxAge *float64
	for _, c := range cache.Hints {

		if c.Scope == "PRIVATE" {
			scope = c.Scope
		}

		if maxAge == nil || *maxAge > c.MaxAge {
			maxAge = &c.MaxAge
		}
	}

	return OverallCachePolicy{
		MaxAge: *maxAge,
		Scope:  scope,
	}
}

func SetCacheHint(ctx context.Context, scope CacheScope, maxAge time.Duration) {
	h := Hint{
		Path:   GetFieldContext(ctx).Path(),
		MaxAge: maxAge.Seconds(),
		Scope:  scope,
	}

	c := GetExtension(ctx, "cacheControl")
	if c == nil {
		cache := &CacheControl{Version: 1}
		cache.AddHint(h)
		RegisterExtension(ctx, "cacheControl", cache)
	}

	if c, ok := c.(*CacheControl); ok {
		c.AddHint(h)
	}
}

func GetOverallCachePolicy(response *Response) (OverallCachePolicy, bool) {
	if cache, ok := response.Extensions["cacheControl"].(*CacheControl); ok {
		overallPolicy := cache.OverallPolicy()
		if overallPolicy.MaxAge > 0 {
			return overallPolicy, true
		}
	}

	return OverallCachePolicy{}, false
}
