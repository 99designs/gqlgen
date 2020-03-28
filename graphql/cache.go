package graphql

import (
	"context"
	"sync"
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

type CacheControlExtension struct {
	Version int    `json:"version"`
	Hints   []Hint `json:"hints"`
	mu      sync.Mutex
}

func (cache *CacheControlExtension) AddHint(h Hint) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.Hints = append(cache.Hints, h)
}

// OverallPolicy return a calculated cache policy
func (cache *CacheControlExtension) OverallPolicy() OverallCachePolicy {
	var (
		scope     = CacheScopePublic
		maxAge    float64
		hasMaxAge bool
	)

	for _, c := range cache.Hints {

		if c.Scope == CacheScopePrivate {
			scope = c.Scope
		}

		if !hasMaxAge || c.MaxAge < maxAge {
			hasMaxAge = true
			maxAge = c.MaxAge
		}
	}

	return OverallCachePolicy{
		MaxAge: maxAge,
		Scope:  scope,
	}
}

const cacheKey = "key"

func WithCacheControlExtension(ctx context.Context) context.Context {
	cache := &CacheControlExtension{Version: 1}
	return context.WithValue(ctx, cacheKey, cache)
}

func CacheControl(ctx context.Context) *CacheControlExtension {
	c := ctx.Value(cacheKey)
	if c, ok := c.(*CacheControlExtension); ok {
		return c
	}

	return nil
}

func SetCacheHint(ctx context.Context, scope CacheScope, maxAge time.Duration) {
	c := ctx.Value(cacheKey)
	if c, ok := c.(*CacheControlExtension); ok {
		c.AddHint(Hint{
			Path:   GetFieldContext(ctx).Path(),
			MaxAge: maxAge.Seconds(),
			Scope:  scope,
		})
	}
}

// GetOverallCachePolicy is responsible to extract cache policy from a Response.
// If does not have any cacheControl in Extensions, it will return (empty, false)
func GetOverallCachePolicy(response *Response) (OverallCachePolicy, bool) {
	if cache, ok := response.Extensions["cacheControl"].(*CacheControlExtension); ok {
		overallPolicy := cache.OverallPolicy()
		if overallPolicy.MaxAge > 0 {
			return overallPolicy, true
		}
	}

	return OverallCachePolicy{}, false
}
