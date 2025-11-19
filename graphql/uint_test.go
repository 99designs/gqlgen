package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalUint(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, uint(0), mustUnmarshalUint(nil))
		assert.Equal(t, uint(123), mustUnmarshalUint(123))
		assert.Equal(t, uint(123), mustUnmarshalUint(int64(123)))
		assert.Equal(t, uint(123), mustUnmarshalUint(json.Number("123")))
		assert.Equal(t, uint(123), mustUnmarshalUint("123"))
	})

	t.Run("can't unmarshal negative numbers", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"negative int", -1, "-1 is an invalid unsigned integer: includes sign"},
			{"negative int64", int64(-1), "-1 is an invalid unsigned integer: includes sign"},
			{
				"negative json.Number",
				json.Number("-1"),
				"-1 is an invalid unsigned integer: includes sign",
			},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&uintSignErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"empty", "", `strconv.ParseUint: parsing "": invalid syntax`},
			{"string", "-1.03", `strconv.ParseUint: parsing "-1.03": invalid syntax`},
			{"json number", json.Number(" 1"), `strconv.ParseUint: parsing " 1": invalid syntax`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				assert.NotErrorAs(t, err, &uintSignErr)
				assert.NotErrorAs(t, err, &intErr)
				assert.Equal(t, uint(0), res)
			})
		}
	})
}

func mustUnmarshalUint(v any) uint {
	res, err := UnmarshalUint(v)
	if err != nil {
		panic(err)
	}
	return res
}

func TestUint8(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalUint8(123)))
		assert.Equal(t, "255", m2s(MarshalUint8(math.MaxUint8)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, uint8(0), mustUnmarshalUint8(nil))
		assert.Equal(t, uint8(123), mustUnmarshalUint8(123))
		assert.Equal(t, uint8(123), mustUnmarshalUint8(int64(123)))
		assert.Equal(t, uint8(123), mustUnmarshalUint8(json.Number("123")))
		assert.Equal(t, uint8(123), mustUnmarshalUint8("123"))
		assert.Equal(t, uint8(255), mustUnmarshalUint8("255"))
	})

	t.Run("can't unmarshal negative numbers", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"negative int", -1, "-1 is an invalid unsigned integer: includes sign"},
			{"negative int64", int64(-1), "-1 is an invalid unsigned integer: includes sign"},
			{
				"negative json.Number",
				json.Number("-1"),
				"-1 is an invalid unsigned integer: includes sign",
			},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint8(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&uintSignErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint8(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var uintSignErr *UintSignError
		var intErr *IntegerError

		res, err := UnmarshalUint8("-1.03")
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \"-1.03\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint8(0), res)

		res, err = UnmarshalUint8(json.Number(" 1"))
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \" 1\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint8(0), res)
	})

	t.Run("overflow", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"int overflow", math.MaxUint8 + 1, "256 overflows unsigned 8-bit integer"},
			{"int64 overflow", int64(math.MaxUint8 + 1), "256 overflows unsigned 8-bit integer"},
			{"json.Number overflow", json.Number("256"), "256 overflows unsigned 8-bit integer"},
			{"string overflow", "256", "256 overflows unsigned 8-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var numberOverflowErr *NumberOverflowError
				var intErr *IntegerError

				res, err := UnmarshalUint8(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&numberOverflowErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint8(0), res)
			})
		}
	})
}

func mustUnmarshalUint8(v any) uint8 {
	res, err := UnmarshalUint8(v)
	if err != nil {
		panic(err)
	}
	return res
}

func TestUint16(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalUint16(123)))
		assert.Equal(t, "65535", m2s(MarshalUint16(math.MaxUint16)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, uint16(0), mustUnmarshalUint16(nil))
		assert.Equal(t, uint16(123), mustUnmarshalUint16(123))
		assert.Equal(t, uint16(123), mustUnmarshalUint16(int64(123)))
		assert.Equal(t, uint16(123), mustUnmarshalUint16(json.Number("123")))
		assert.Equal(t, uint16(123), mustUnmarshalUint16("123"))
		assert.Equal(t, uint16(65535), mustUnmarshalUint16("65535"))
	})

	t.Run("can't unmarshal negative numbers", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"negative int", -1, "-1 is an invalid unsigned integer: includes sign"},
			{"negative int64", int64(-1), "-1 is an invalid unsigned integer: includes sign"},
			{
				"negative json.Number",
				json.Number("-1"),
				"-1 is an invalid unsigned integer: includes sign",
			},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint16(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&uintSignErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint16(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var uintSignErr *UintSignError
		var intErr *IntegerError

		res, err := UnmarshalUint16("-1.03")
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \"-1.03\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint16(0), res)

		res, err = UnmarshalUint16(json.Number(" 1"))
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \" 1\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint16(0), res)
	})

	t.Run("overflow", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"int overflow", math.MaxUint16 + 1, "65536 overflows unsigned 16-bit integer"},
			{
				"int64 overflow",
				int64(math.MaxUint16 + 1),
				"65536 overflows unsigned 16-bit integer",
			},
			{
				"json.Number overflow",
				json.Number("65536"),
				"65536 overflows unsigned 16-bit integer",
			},
			{"string overflow", "65536", "65536 overflows unsigned 16-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var numberOverflowErr *NumberOverflowError
				var intErr *IntegerError

				res, err := UnmarshalUint16(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&numberOverflowErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint16(0), res)
			})
		}
	})
}

func mustUnmarshalUint16(v any) uint16 {
	res, err := UnmarshalUint16(v)
	if err != nil {
		panic(err)
	}
	return res
}

func TestUint32(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalUint32(123)))
		assert.Equal(t, "4294967295", m2s(MarshalUint32(math.MaxUint32)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, uint32(0), mustUnmarshalUint32(nil))
		assert.Equal(t, uint32(123), mustUnmarshalUint32(123))
		assert.Equal(t, uint32(123), mustUnmarshalUint32(int64(123)))
		assert.Equal(t, uint32(123), mustUnmarshalUint32(json.Number("123")))
		assert.Equal(t, uint32(123), mustUnmarshalUint32("123"))
		assert.Equal(t, uint32(4294967295), mustUnmarshalUint32("4294967295"))
	})

	t.Run("can't unmarshal negative numbers", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"negative int", -1, "-1 is an invalid unsigned integer: includes sign"},
			{"negative int64", int64(-1), "-1 is an invalid unsigned integer: includes sign"},
			{
				"negative json.Number",
				json.Number("-1"),
				"-1 is an invalid unsigned integer: includes sign",
			},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint32(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&uintSignErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint32(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var uintSignErr *UintSignError
		var intErr *IntegerError

		res, err := UnmarshalUint32("-1.03")
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \"-1.03\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint32(0), res)

		res, err = UnmarshalUint32(json.Number(" 1"))
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \" 1\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint32(0), res)
	})

	t.Run("overflow", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"int overflow", math.MaxUint32 + 1, "4294967296 overflows unsigned 32-bit integer"},
			{
				"int64 overflow",
				int64(math.MaxUint32 + 1),
				"4294967296 overflows unsigned 32-bit integer",
			},
			{
				"json.Number overflow",
				json.Number("4294967296"),
				"4294967296 overflows unsigned 32-bit integer",
			},
			{"string overflow", "4294967296", "4294967296 overflows unsigned 32-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var numberOverflowErr *NumberOverflowError
				var intErr *IntegerError

				res, err := UnmarshalUint32(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&numberOverflowErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint32(0), res)
			})
		}
	})
}

func mustUnmarshalUint32(v any) uint32 {
	res, err := UnmarshalUint32(v)
	if err != nil {
		panic(err)
	}
	return res
}

func TestUint64(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		assert.Equal(t, "123", m2s(MarshalUint64(123)))
	})

	t.Run("unmarshal", func(t *testing.T) {
		assert.Equal(t, uint64(0), mustUnmarshalUint64(nil))
		assert.Equal(t, uint64(123), mustUnmarshalUint64(123))
		assert.Equal(t, uint64(123), mustUnmarshalUint64(int64(123)))
		assert.Equal(t, uint64(123), mustUnmarshalUint64(json.Number("123")))
		assert.Equal(t, uint64(123), mustUnmarshalUint64("123"))
	})

	t.Run("can't unmarshal negative numbers", func(t *testing.T) {
		cases := []struct {
			name string
			v    any
			err  string
		}{
			{"negative int", -1, "-1 is an invalid unsigned integer: includes sign"},
			{"negative int64", int64(-1), "-1 is an invalid unsigned integer: includes sign"},
			{
				"negative json.Number",
				json.Number("-1"),
				"-1 is an invalid unsigned integer: includes sign",
			},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint64(tc.v)
				require.EqualError(
					t,
					err,
					tc.err,
				)
				require.ErrorAs(
					t,
					err,
					&uintSignErr,
				)
				require.ErrorAs(
					t,
					err,
					&intErr,
				)
				assert.Equal(t, uint64(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var uintSignErr *UintSignError
		var intErr *IntegerError

		res, err := UnmarshalUint64("-1.03")
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \"-1.03\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint64(0), res)

		res, err = UnmarshalUint64(json.Number(" 1"))
		require.EqualError(
			t,
			err,
			"strconv.ParseUint: parsing \" 1\": invalid syntax",
		)
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint64(0), res)
	})
}

func mustUnmarshalUint64(v any) uint64 {
	res, err := UnmarshalUint64(v)
	if err != nil {
		panic(err)
	}
	return res
}

func beforeUnmarshalUint(v any) (uint, error) {
	switch v := v.(type) {
	case string:
		u64, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			var strconvErr *strconv.NumError
			if errors.As(err, &strconvErr) && isSignedInteger(v) {
				return 0, newUintSignError(v)
			}
			return 0, err
		}
		return uint(u64), err
	case int:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(int64(v), 10))
		}
		return uint(v), nil
	case int64:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(v, 10))
		}
		return uint(v), nil
	case json.Number:
		u64, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			var strconvErr *strconv.NumError
			if errors.As(err, &strconvErr) && isSignedInteger(string(v)) {
				return 0, newUintSignError(string(v))
			}
			return 0, err
		}
		return uint(u64), err
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not an uint", v)
	}
}

func BenchmarkUnmarshalUintInitial(b *testing.B) {
	numbers := makeRandomNumberSlice(false)

	for range b.N {
		for i := range numbers {
			_, _ = beforeUnmarshalUint(numbers[i])
		}
	}
}

func BenchmarkUnmarshalUintNew(b *testing.B) {
	numbers := makeRandomNumberSlice(false)

	for range b.N {
		for i := range numbers {
			_, _ = UnmarshalUint(numbers[i])
		}
	}
}
