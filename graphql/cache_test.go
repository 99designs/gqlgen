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

func TestMapCacheEdgeCases(t *testing.T) {
	type testCase struct {
		name       string
		key        string
		value      string
		initialVal string // Initial value if needed (for overwrite tests)
		wantValue  string
		wantOk     bool
	}

	tests := []testCase{
		{
			name:      "Empty Key",
			key:       "",
			value:     "valueForEmptyKey",
			wantValue: "valueForEmptyKey",
			wantOk:    true,
		},
		{
			name:      "Very Long Key",
			key:       "key" + string(make([]rune, 10000)),
			value:     "valueForLongKey",
			wantValue: "valueForLongKey",
			wantOk:    true,
		},
		{
			name:       "Overwrite Existing Key",
			key:        "testKey",
			initialVal: "initialValue",
			value:      "newValue",
			wantValue:  "newValue",
			wantOk:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cache := MapCache{}
			ctx := context.Background()

			// Set initial value if needed
			if tc.initialVal != "" {
				cache.Add(ctx, tc.key, tc.initialVal)
			}

			// Add the main value
			cache.Add(ctx, tc.key, tc.value)

			// Test Get
			gotValue, ok := cache.Get(ctx, tc.key)
			assert.Equal(t, tc.wantOk, ok, "Expected ok to be %v", tc.wantOk)
			assert.Equal(t, tc.wantValue, gotValue, "Expected value to be %v", tc.wantValue)
		})
	}
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
		for key := range entries {
			gotValue, ok := cache.Get(ctx, key)
			assert.False(t, ok, "Get should not find the key %s", key)
			assert.Nil(t, gotValue, "Get should return nil for key %s", key)
		}
	})
}

func TestNoCacheEdgeCases(t *testing.T) {
	type testCase struct {
		name      string
		key       string
		value     string
		wantOk    bool
		wantValue any
	}

	tests := []testCase{
		{
			name:      "Get After Add",
			key:       "anyKey",
			value:     "anyValue",
			wantOk:    false,
			wantValue: nil,
		},
		{
			name:      "Empty Key",
			key:       "",
			value:     "value",
			wantOk:    false,
			wantValue: nil,
		},
		{
			name:      "Very Long Key",
			key:       "key" + string(make([]rune, 10000)),
			value:     "value",
			wantOk:    false,
			wantValue: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cache := NoCache{}
			ctx := context.Background()

			// Test Add
			cache.Add(ctx, tc.key, tc.value)

			// Test Get
			gotValue, ok := cache.Get(ctx, tc.key)
			assert.Equal(t, tc.wantOk, ok, "Get should not find the key")
			assert.Equal(t, tc.wantValue, gotValue, "Get should return nil for any key")
		})
	}
}
