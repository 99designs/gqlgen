package graphql

import "context"

// Cache is a shared store for APQ and query AST caching
type Cache interface {
	// Get looks up a key's value from the cache.
	Get(ctx context.Context, key string) (value any, ok bool)

	// Add adds a value to the cache.
	Add(ctx context.Context, key string, value any)
}

// MapCache is the simplest implementation of a cache, because it can not evict it should only be used in tests
type MapCache map[string]any

// Get looks up a key's value from the cache.
func (m MapCache) Get(_ context.Context, key string) (value any, ok bool) {
	v, ok := m[key]
	return v, ok
}

// Add adds a value to the cache.
func (m MapCache) Add(_ context.Context, key string, value any) { m[key] = value }

type NoCache struct{}

func (n NoCache) Get(_ context.Context, _ string) (value any, ok bool) { return nil, false }
func (n NoCache) Add(_ context.Context, _ string, _ any)               {}
