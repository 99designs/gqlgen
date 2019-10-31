package graphql

// Cache is a shared store for APQ and query AST caching
type Cache interface {
	// Get looks up a key's value from the cache.
	Get(key string) (value interface{}, ok bool)

	// Add adds a value to the cache.
	Add(key string, value interface{})
}

// MapCache is the simplest implementation of a cache, because it can not evict it should only be used in tests
type MapCache map[string]interface{}

// Get looks up a key's value from the cache.
func (m MapCache) Get(key string) (value interface{}, ok bool) {
	v, ok := m[key]
	return v, ok
}

// Add adds a value to the cache.
func (m MapCache) Add(key string, value interface{}) { m[key] = value }

type NoCache struct{}

func (n NoCache) Get(key string) (value interface{}, ok bool) { return nil, false }
func (n NoCache) Add(key string, value interface{})           {}
