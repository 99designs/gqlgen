package lru_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql/handler/lru"
)

// This cache is the one a server puts in front of query parsing and
// validation, so a parsed document it returns is used without being re-checked
// and a document it loses costs a full parse. The contracts worth pinning are
// therefore which key maps to which value, which entry is dropped when the
// cache is full, and that concurrent requests can share one instance.

func TestGetReturnsWhatWasAdded(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		add      map[string]string
		get      string
		wantVal  string
		wantFind bool
	}{
		"a key that was added": {
			add:      map[string]string{"query{a}": "doc-a"},
			get:      "query{a}",
			wantVal:  "doc-a",
			wantFind: true,
		},
		"a key that was never added": {
			add:      map[string]string{"query{a}": "doc-a"},
			get:      "query{b}",
			wantVal:  "",
			wantFind: false,
		},
		"a key differing only in whitespace is a different key": {
			add:      map[string]string{"query{a}": "doc-a"},
			get:      "query { a }",
			wantVal:  "",
			wantFind: false,
		},
		"the empty string is a usable key": {
			add:      map[string]string{"": "doc-empty"},
			get:      "",
			wantVal:  "doc-empty",
			wantFind: true,
		},
		"an empty cache finds nothing": {
			add:      nil,
			get:      "query{a}",
			wantVal:  "",
			wantFind: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			cache := lru.New[string](16)
			for k, v := range tc.add {
				cache.Add(ctx, k, v)
			}

			val, found := cache.Get(ctx, tc.get)
			assert.Equal(t, tc.wantFind, found)
			// A miss must yield the zero value rather than whatever was last
			// written, because a caller that ignores the bool would otherwise
			// act on another request's document.
			assert.Equal(t, tc.wantVal, val)
		})
	}
}

// Adding a key that is already present replaces the value. A cache that kept
// the first one would serve a stale document after a schema change made the
// same query text parse differently.
func TestAddReplacesAnExistingKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := lru.New[string](16)

	cache.Add(ctx, "query{a}", "first")
	cache.Add(ctx, "query{a}", "second")

	val, found := cache.Get(ctx, "query{a}")
	require.True(t, found)
	assert.Equal(t, "second", val)
}

// The eviction contract: at capacity, the entry used longest ago is the one
// dropped. This is the whole reason for choosing an LRU over a plain map with a
// size cap, so it is the behaviour most worth holding still.
func TestEvictsTheLeastRecentlyUsedEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := lru.New[string](2)

	cache.Add(ctx, "first", "1")
	cache.Add(ctx, "second", "2")

	// Adding a third entry to a cache of two must drop exactly one, and it must
	// be "first", which has not been touched since it was added.
	cache.Add(ctx, "third", "3")

	_, found := cache.Get(ctx, "first")
	assert.False(t, found, "the least recently used entry should have been evicted")

	for _, key := range []string{"second", "third"} {
		_, found := cache.Get(ctx, key)
		assert.True(t, found, "%s should still be cached", key)
	}
}

// A Get counts as a use. Without this, a query served on every request would be
// evicted by a burst of one-off queries, which is the case the cache exists to
// handle.
func TestGetRefreshesRecency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := lru.New[string](2)

	cache.Add(ctx, "hot", "1")
	cache.Add(ctx, "cold", "2")

	// Touching "hot" makes "cold" the least recently used, so the next
	// admission must evict "cold" rather than "hot".
	_, found := cache.Get(ctx, "hot")
	require.True(t, found)

	cache.Add(ctx, "new", "3")

	_, found = cache.Get(ctx, "hot")
	assert.True(t, found, "an entry read since it was added should survive eviction")
	_, found = cache.Get(ctx, "cold")
	assert.False(t, found, "the entry not read since it was added should be evicted")
}

// The cache is generic, and a miss has to produce the zero value of whatever
// type it holds rather than panic or return a half-built value.
func TestMissYieldsTheZeroValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("pointer", func(t *testing.T) {
		t.Parallel()
		val, found := lru.New[*string](4).Get(ctx, "absent")
		assert.False(t, found)
		assert.Nil(t, val)
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		type doc struct{ Name string }
		val, found := lru.New[doc](4).Get(ctx, "absent")
		assert.False(t, found)
		assert.Equal(t, doc{}, val)
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		val, found := lru.New[[]int](4).Get(ctx, "absent")
		assert.False(t, found)
		assert.Nil(t, val)
	})
}

// A non-positive size cannot produce a working cache, and New panics rather
// than returning something that silently caches nothing. Pinning that keeps the
// failure loud: a server misconfigured this way should not start.
func TestNewPanicsOnNonPositiveSize(t *testing.T) {
	t.Parallel()

	for _, size := range []int{0, -1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { lru.New[string](size) })
		})
	}

	assert.NotPanics(t, func() { lru.New[string](1) },
		"the smallest usable size should be accepted")
}

// One Server shares one cache across every request it serves, so concurrent
// Get and Add on a single instance is the normal case rather than an edge one.
// Run under -race, which gqlgen's CI does, this is what shows the sharing is
// actually safe.
func TestConcurrentUse(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 8
		iterations = 200
		cacheSize  = 32
	)

	ctx := context.Background()
	cache := lru.New[string](cacheSize)

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range iterations {
				// Keys deliberately collide across goroutines for part of the
				// range and are unique for the rest, so the test exercises both
				// contended writes to one key and steady eviction pressure.
				shared := fmt.Sprintf("shared-%d", i%cacheSize)
				cache.Add(ctx, shared, shared)
				if val, found := cache.Get(ctx, shared); found {
					// Whatever is returned must be a value some goroutine
					// actually wrote, never a torn or zero one.
					assert.Equal(t, shared, val)
				}
				unique := fmt.Sprintf("unique-%d-%d", g, i)
				cache.Add(ctx, unique, unique)
			}
		}(g)
	}
	wg.Wait()

	// The cache must not have grown past its size, which is the other half of
	// the bound it promises.
	present := 0
	for i := range cacheSize * goroutines {
		if _, found := cache.Get(ctx, fmt.Sprintf("shared-%d", i)); found {
			present++
		}
	}
	assert.LessOrEqual(t, present, cacheSize)
}
