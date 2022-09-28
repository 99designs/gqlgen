package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.Equal(t, "123", mustUnmarshalString("123"))
		assert.Equal(t, "123", mustUnmarshalString(123))
		assert.Equal(t, "123", mustUnmarshalString(int64(123)))
		assert.Equal(t, "123", mustUnmarshalString(float64(123)))
		assert.Equal(t, "123", mustUnmarshalString(json.Number("123")))
		assert.Equal(t, "true", mustUnmarshalString(true))
		assert.Equal(t, "false", mustUnmarshalString(false))
		assert.Equal(t, "null", mustUnmarshalString(nil))
	})
}

func mustUnmarshalString(v interface{}) string {
	res, err := UnmarshalString(v)
	if err != nil {
		panic(err)
	}
	return res
}
