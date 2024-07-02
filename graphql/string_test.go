package graphql

import (
	"testing"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, `"hello"`, m2s(MarshalString("hello")))
		assert.Equal(t, `"he\tllo"`, m2s(MarshalString("he\tllo")))
		assert.Equal(t, `"he\tllo"`, m2s(MarshalString("he	llo")))
		assert.Equal(t, `"he\nllo"`, m2s(MarshalString("he\nllo")))
		assert.Equal(t, `"he\r\nllo"`, m2s(MarshalString("he\r\nllo")))
		assert.Equal(t, `"he\\llo"`, m2s(MarshalString(`he\llo`)))
		assert.Equal(t, `"quotes\"nested\"in\"quotes\""`, m2s(MarshalString(`quotes"nested"in"quotes"`)))
		assert.Equal(t, `"\u0000"`, m2s(MarshalString("\u0000")))
		assert.Equal(t, `"\u0000"`, m2s(MarshalString("\u0000")))
		assert.Equal(t, "\"\U000fe4ed\"", m2s(MarshalString("\U000fe4ed")))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, "123", mustUnmarshalString(t, "123"))
		assert.Equal(t, "123", mustUnmarshalString(t, 123))
		assert.Equal(t, "123", mustUnmarshalString(t, int64(123)))
		assert.Equal(t, "123", mustUnmarshalString(t, float64(123)))
		assert.Equal(t, "123", mustUnmarshalString(t, json.Number("123")))
		assert.Equal(t, "true", mustUnmarshalString(t, true))
		assert.Equal(t, "false", mustUnmarshalString(t, false))
		assert.Equal(t, "", mustUnmarshalString(t, nil))
	})
}

func mustUnmarshalString(t *testing.T, v any) string {
	res, err := UnmarshalString(v)
	require.NoError(t, err)
	return res
}
