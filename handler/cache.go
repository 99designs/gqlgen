package handler

// Cache for the graphqlHandler
type Cache interface {
	// Get looks up a key's value from the cache.
	Get(key interface{}) (value interface{}, ok bool)
	// Add adds a value to the cache.  Returns true if an eviction occurred.
	Add(key, value interface{}) (evicted bool)
}
