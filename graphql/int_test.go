package graphql

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInt8(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt8(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, int8(0), mustUnmarshalInt8(t, nil))
		assert.Equal(t, int8(123), mustUnmarshalInt8(t, 123))
		assert.Equal(t, int8(123), mustUnmarshalInt8(t, int64(123)))
		assert.Equal(t, int8(123), mustUnmarshalInt8(t, json.Number("123")))
		assert.Equal(t, int8(123), mustUnmarshalInt8(t, "123"))
		assert.Equal(t, int8(0), mustUnmarshalInt8(t, nil))
	})

	t.Run("overflow", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"positive int overflow", math.MaxInt8 + 1, "128 overflows signed 8-bit integer"},
			{"negative int overflow", math.MinInt8 - 1, "-129 overflows signed 8-bit integer"},
			{"positive int64 overflow", int64(math.MaxInt8 + 1), "128 overflows signed 8-bit integer"},
			{"negative int64 overflow", int64(math.MinInt8 - 1), "-129 overflows signed 8-bit integer"},
			{"positive json.Number overflow", json.Number("128"), "128 overflows signed 8-bit integer"},
			{"negative json.Number overflow", json.Number("-129"), "-129 overflows signed 8-bit integer"},
			{"positive string overflow", "128", "128 overflows signed 8-bit integer"},
			{"negative string overflow", "-129", "-129 overflows signed 8-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var numberOverflowErr *NumberOverflowError
				var intErr *IntegerError

				res, err := UnmarshalInt8(tc.v)
				assert.EqualError(t, err, tc.err)          //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &numberOverflowErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)            //nolint:testifylint // An error assertion makes more sense.
				assert.Equal(t, int8(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var intErr *IntegerError

		res, err := UnmarshalInt8("-1.03")
		assert.EqualError(t, err, "strconv.ParseInt: parsing \"-1.03\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, int8(0), res)

		res, err = UnmarshalInt8(json.Number(" 1"))
		assert.EqualError(t, err, "strconv.ParseInt: parsing \" 1\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, int8(0), res)
	})
}

func mustUnmarshalInt8(t *testing.T, v any) int8 {
	res, err := UnmarshalInt8(v)
	require.NoError(t, err)
	return res
}

func TestInt16(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt16(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, int16(0), mustUnmarshalInt16(t, nil))
		assert.Equal(t, int16(123), mustUnmarshalInt16(t, 123))
		assert.Equal(t, int16(123), mustUnmarshalInt16(t, int64(123)))
		assert.Equal(t, int16(123), mustUnmarshalInt16(t, json.Number("123")))
		assert.Equal(t, int16(123), mustUnmarshalInt16(t, "123"))
		assert.Equal(t, int16(0), mustUnmarshalInt16(t, nil))
	})

	t.Run("overflow", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"positive int overflow", math.MaxInt16 + 1, "32768 overflows signed 16-bit integer"},
			{"negative int overflow", math.MinInt16 - 1, "-32769 overflows signed 16-bit integer"},
			{"positive int64 overflow", int64(math.MaxInt16 + 1), "32768 overflows signed 16-bit integer"},
			{"negative int64 overflow", int64(math.MinInt16 - 1), "-32769 overflows signed 16-bit integer"},
			{"positive json.Number overflow", json.Number("32768"), "32768 overflows signed 16-bit integer"},
			{"negative json.Number overflow", json.Number("-32769"), "-32769 overflows signed 16-bit integer"},
			{"positive string overflow", "32768", "32768 overflows signed 16-bit integer"},
			{"negative string overflow", "-32769", "-32769 overflows signed 16-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var numberOverflowErr *NumberOverflowError
				var intErr *IntegerError

				res, err := UnmarshalInt16(tc.v)
				assert.EqualError(t, err, tc.err)          //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &numberOverflowErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)            //nolint:testifylint // An error assertion makes more sense.
				assert.Equal(t, int16(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var intErr *IntegerError

		res, err := UnmarshalInt16("-1.03")
		assert.EqualError(t, err, "strconv.ParseInt: parsing \"-1.03\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, int16(0), res)

		res, err = UnmarshalInt16(json.Number(" 1"))
		assert.EqualError(t, err, "strconv.ParseInt: parsing \" 1\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, int16(0), res)
	})
}

func mustUnmarshalInt16(t *testing.T, v any) int16 {
	res, err := UnmarshalInt16(v)
	require.NoError(t, err)
	return res
}

func TestInt(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, 0, mustUnmarshalInt(t, nil))
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
		assert.Equal(t, int32(0), mustUnmarshalInt32(t, nil))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, 123))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, int64(123)))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, json.Number("123")))
		assert.Equal(t, int32(123), mustUnmarshalInt32(t, "123"))
		assert.Equal(t, int32(0), mustUnmarshalInt32(t, nil))
	})

	t.Run("overflow", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"positive int overflow", math.MaxInt32 + 1, "2147483648 overflows signed 32-bit integer"},
			{"negative int overflow", math.MinInt32 - 1, "-2147483649 overflows signed 32-bit integer"},
			{"positive int64 overflow", int64(math.MaxInt32 + 1), "2147483648 overflows signed 32-bit integer"},
			{"negative int64 overflow", int64(math.MinInt32 - 1), "-2147483649 overflows signed 32-bit integer"},
			{"positive json.Number overflow", json.Number("2147483648"), "2147483648 overflows signed 32-bit integer"},
			{"negative json.Number overflow", json.Number("-2147483649"), "-2147483649 overflows signed 32-bit integer"},
			{"positive string overflow", "2147483648", "2147483648 overflows signed 32-bit integer"},
			{"negative string overflow", "-2147483649", "-2147483649 overflows signed 32-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var numberOverflowErr *NumberOverflowError
				var intErr *IntegerError

				res, err := UnmarshalInt32(tc.v)
				assert.EqualError(t, err, tc.err)          //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &numberOverflowErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)            //nolint:testifylint // An error assertion makes more sense.
				assert.Equal(t, int32(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var intErr *IntegerError

		res, err := UnmarshalInt32("-1.03")
		assert.EqualError(t, err, "strconv.ParseInt: parsing \"-1.03\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, int32(0), res)

		res, err = UnmarshalInt32(json.Number(" 1"))
		assert.EqualError(t, err, "strconv.ParseInt: parsing \" 1\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, int32(0), res)
	})
}

func mustUnmarshalInt32(t *testing.T, v any) int32 {
	res, err := UnmarshalInt32(v)
	require.NoError(t, err)
	return res
}

func TestInt64(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalInt64(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, int64(0), mustUnmarshalInt64(t, nil))
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
