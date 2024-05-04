package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapCache(t *testing.T) {
	t.Run("Add and Get", func(t *testing.T) {
		cache := MapCache{}
		ctx := context.Background()
		key := "testKey"
		value := "testValue"

		// Test Add
		cache.Add(ctx, key, value)
		val, ok := cache[key]
		assert.True(t, ok, "Key should exist in cache")
		assert.Equal(t, value, val, "Cache should return the correct value for a key")

		// Test Get
		gotValue, ok := cache.Get(ctx, key)
		assert.True(t, ok, "Get should find the key")
		assert.Equal(t, value, gotValue, "Get should return the correct value")
	})
}

func TestMapCacheMultipleEntries(t *testing.T) {
	t.Run("Multiple Add and Get", func(t *testing.T) {
		cache := MapCache{}
		ctx := context.Background()

		// Define multiple key-value pairs
		entries := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		// Test Add for multiple entries
		for key, value := range entries {
			cache.Add(ctx, key, value)
			val, ok := cache[key]
			assert.True(t, ok, "Key %s should exist in cache", key)
			assert.Equal(t, value, val, "Cache should return the correct value for key %s", key)
		}

		// Test Get for multiple entries
		for key, expectedValue := range entries {
			gotValue, ok := cache.Get(ctx, key)
			assert.True(t, ok, "Get should find the key %s", key)
			assert.Equal(t, expectedValue, gotValue, "Get should return the correct value for key %s", key)
		}
	})
}

func TestNoCache(t *testing.T) {
	t.Run("Add and Get", func(t *testing.T) {
		cache := NoCache{}
		ctx := context.Background()
		key := "testKey"
		value := "testValue"

		// Test Add
		cache.Add(ctx, key, value) // Should do nothing

		// Test Get
		gotValue, ok := cache.Get(ctx, key)
		assert.False(t, ok, "Get should not find the key")
		assert.Nil(t, gotValue, "Get should return nil for any key")
	})
}

func TestNoCacheMultipleEntries(t *testing.T) {
	t.Run("Multiple Add and Get", func(t *testing.T) {
		cache := NoCache{}
		ctx := context.Background()

		// Define multiple key-value pairs
		entries := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		// Test Add for multiple entries
		for key, value := range entries {
			cache.Add(ctx, key, value) // Should do nothing
		}

		// Test Get for multiple entries
		for key, _ := range entries {
			gotValue, ok := cache.Get(ctx, key)
			assert.False(t, ok, "Get should not find the key %s", key)
			assert.Nil(t, gotValue, "Get should return nil for key %s", key)
		}
	})
}
