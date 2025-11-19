package graphql

import (
	"hash/fnv"
	"reflect"
	"sync"

	"github.com/vektah/gqlparser/v2/ast"
)

// collectFieldsCacheKey is the cache key for CollectFields results.
type collectFieldsCacheKey struct {
	selectionPtr  uintptr // Pointer to the underlying SelectionSet data
	selectionLen  int     // Length of the selection set
	satisfiesHash uint64  // Hash of the satisfies array
}

// collectFieldsCacheStore manages CollectFields cache entries safely.
type collectFieldsCacheStore struct {
	mu    sync.RWMutex
	items map[collectFieldsCacheKey][]CollectedField
}

// Get returns the cached result for the key if present.
func (s *collectFieldsCacheStore) Get(key collectFieldsCacheKey) ([]CollectedField, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.items == nil {
		return nil, false
	}
	val, ok := s.items[key]
	return val, ok
}

// Add stores the value when absent and returns the cached value.
func (s *collectFieldsCacheStore) Add(
	key collectFieldsCacheKey,
	value []CollectedField,
) []CollectedField {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[collectFieldsCacheKey][]CollectedField)
	}

	if existing, ok := s.items[key]; ok {
		return existing
	}
	s.items[key] = value
	return value
}

// Len returns the number of cached entries.
func (s *collectFieldsCacheStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// makeCollectFieldsCacheKey generates a cache key for CollectFields.
func makeCollectFieldsCacheKey(selSet ast.SelectionSet, satisfies []string) collectFieldsCacheKey {
	var selectionPtr uintptr
	if selSet != nil {
		selectionPtr = reflect.ValueOf(selSet).Pointer()
	}

	h := fnv.New64a()
	for _, s := range satisfies {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}

	return collectFieldsCacheKey{
		selectionPtr:  selectionPtr,
		selectionLen:  len(selSet),
		satisfiesHash: h.Sum64(),
	}
}
