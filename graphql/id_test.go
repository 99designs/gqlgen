package graphql

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalID(t *testing.T) {
	marshalID := func(s string) string {
		var buf bytes.Buffer
		MarshalID(s).MarshalGQL(&buf)
		return buf.String()
	}

	assert.Equal(t, `"hello"`, marshalID("hello"))
	assert.Equal(t, `"he\tllo"`, marshalID("he\tllo"))
	assert.Equal(t, `"he\tllo"`, marshalID("he	llo"))
	assert.Equal(t, `"he\nllo"`, marshalID("he\nllo"))
	assert.Equal(t, `"he\r\nllo"`, marshalID("he\r\nllo"))
	assert.Equal(t, `"he\\llo"`, marshalID(`he\llo`))
	assert.Equal(t, `"quotes\"nested\"in\"quotes\""`, marshalID(`quotes"nested"in"quotes"`))
	assert.Equal(t, `"\u0000"`, marshalID("\u0000"))
	assert.Equal(t, "\"\U000fe4ed\"", marshalID("\U000fe4ed"))
	assert.Equal(t, "\"\\u001B\"", marshalID("\u001B"))
}

func TestUnmarshalID(t *testing.T) {
	tests := []struct {
		Name        string
		Input       any
		Expected    string
		ShouldError bool
	}{
		{
			Name:     "string",
			Input:    "str",
			Expected: "str",
		},
		{
			Name:     "json.Number float64",
			Input:    json.Number("1.2"),
			Expected: "1.2",
		},
		{
			Name:     "int64",
			Input:    int64(12),
			Expected: "12",
		},
		{
			Name:     "int64 max",
			Input:    math.MaxInt64,
			Expected: "9223372036854775807",
		},
		{
			Name:     "int64 min",
			Input:    math.MinInt64,
			Expected: "-9223372036854775808",
		},
		{
			Name:     "bool true",
			Input:    true,
			Expected: "true",
		},
		{
			Name:     "bool false",
			Input:    false,
			Expected: "false",
		},
		{
			Name:     "nil",
			Input:    nil,
			Expected: "null",
		},
		{
			Name:     "float64",
			Input:    1.234567,
			Expected: "1.234567",
		},
		{
			Name:     "float64 0",
			Input:    0.0,
			Expected: "0.000000",
		},
		{
			Name:     "float64 loss of precision",
			Input:    0.0000005,
			Expected: "0.000000",
		},
		{
			Name:     "float64 rounding up",
			Input:    0.0000006,
			Expected: "0.000001",
		},
		{
			Name:     "float64 negative",
			Input:    -1.234560,
			Expected: "-1.234560",
		},
		{
			Name:     "float64 math.Inf(0)",
			Input:    math.Inf(0),
			Expected: "+Inf",
		},
		{
			Name:     "float64 math.Inf(-1)",
			Input:    math.Inf(-1),
			Expected: "-Inf",
		},
		{
			Name:     "float64 -math.Inf(0)",
			Input:    -math.Inf(0),
			Expected: "-Inf",
		},
		{
			Name:        "not a string",
			Input:       struct{}{},
			ShouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			id, err := UnmarshalID(tt.Input)

			assert.Equal(t, tt.Expected, id)
			if tt.ShouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMarshalUintID(t *testing.T) {
	assert.Equal(t, `"12"`, m2s(MarshalUintID(12)))
}

func TestUnMarshalUintID(t *testing.T) {
	result, err := UnmarshalUintID("12")
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)

	result, err = UnmarshalUintID(12)
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)

	result, err = UnmarshalUintID(int64(12))
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)

	result, err = UnmarshalUintID(int32(12))
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)

	result, err = UnmarshalUintID(int(12))
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)

	result, err = UnmarshalUintID(uint32(12))
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)

	result, err = UnmarshalUintID(uint64(12))
	assert.Equal(t, uint(12), result)
	require.NoError(t, err)
}
