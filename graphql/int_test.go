package graphql

import (
	"github.com/goccy/go-json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInt(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, 123, mustUnmarshalInt(t, 123))
		assert.Equal(t, 123, mustUnmarshalInt(t, int64(123)))
		assert.Equal(t, 123, mustUnmarshalInt(t, json.Number("123")))
		assert.Equal(t, 123, mustUnmarshalInt(t, "123"))
		assert.Equal(t, 0, mustUnmarshalInt(t, nil))
	})
}

func mustUnmarshalInt(t *testing.T, v any) int {
	res, err := UnmarshalInt(v)
	require.NoError(t, err)
	return res
}

func TestInt32(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt32(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, 123))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, int64(123)))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, json.Number("123")))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, "123"))
		assert.Equal(t, int32(0), mustUnmarshalInt32(t, nil))
	})
}

func mustUnmarshalInt32(t *testing.T, v any) int32 {
	res, err := UnmarshalInt32(v)
	require.NoError(t, err)
	return res
}

func TestInt64(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt32(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, int64(123), mustUnmarshalInt64(t, 123))
		assert.Equal(t, int64(123), mustUnmarshalInt64(t, int64(123)))
		assert.Equal(t, int64(123), mustUnmarshalInt64(t, json.Number("123")))
		assert.Equal(t, int64(123), mustUnmarshalInt64(t, "123"))
		assert.Equal(t, int64(0), mustUnmarshalInt64(t, nil))
	})
}

func mustUnmarshalInt64(t *testing.T, v any) int64 {
	res, err := UnmarshalInt64(v)
	require.NoError(t, err)
	return res
}
