package graphql

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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
			{"negative json.Number", json.Number("-1"), "-1 is an invalid unsigned integer: includes sign"},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint(tc.v)
				assert.EqualError(t, err, tc.err)    //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &uintSignErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)      //nolint:testifylint // An error assertion makes more sense.
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
				assert.EqualError(t, err, tc.err) //nolint:testifylint // An error assertion makes more sense.
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
			{"negative json.Number", json.Number("-1"), "-1 is an invalid unsigned integer: includes sign"},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint32(tc.v)
				assert.EqualError(t, err, tc.err)    //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &uintSignErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)      //nolint:testifylint // An error assertion makes more sense.
				assert.Equal(t, uint32(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var uintSignErr *UintSignError
		var intErr *IntegerError

		res, err := UnmarshalUint32("-1.03")
		assert.EqualError(t, err, "strconv.ParseUint: parsing \"-1.03\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint32(0), res)

		res, err = UnmarshalUint32(json.Number(" 1"))
		assert.EqualError(t, err, "strconv.ParseUint: parsing \" 1\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
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
			{"int64 overflow", int64(math.MaxUint32 + 1), "4294967296 overflows unsigned 32-bit integer"},
			{"json.Number overflow", json.Number("4294967296"), "4294967296 overflows unsigned 32-bit integer"},
			{"string overflow", "4294967296", "4294967296 overflows unsigned 32-bit integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uint32OverflowErr *Uint32OverflowError
				var intErr *IntegerError

				res, err := UnmarshalUint32(tc.v)
				assert.EqualError(t, err, tc.err)          //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &uint32OverflowErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)            //nolint:testifylint // An error assertion makes more sense.
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
			{"negative json.Number", json.Number("-1"), "-1 is an invalid unsigned integer: includes sign"},
			{"negative string", "-1", "-1 is an invalid unsigned integer: includes sign"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var uintSignErr *UintSignError
				var intErr *IntegerError

				res, err := UnmarshalUint64(tc.v)
				assert.EqualError(t, err, tc.err)    //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &uintSignErr) //nolint:testifylint // An error assertion makes more sense.
				assert.ErrorAs(t, err, &intErr)      //nolint:testifylint // An error assertion makes more sense.
				assert.Equal(t, uint64(0), res)
			})
		}
	})

	t.Run("invalid string numbers are not integer errors", func(t *testing.T) {
		var uintSignErr *UintSignError
		var intErr *IntegerError

		res, err := UnmarshalUint64("-1.03")
		assert.EqualError(t, err, "strconv.ParseUint: parsing \"-1.03\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
		assert.NotErrorAs(t, err, &uintSignErr)
		assert.NotErrorAs(t, err, &intErr)
		assert.Equal(t, uint64(0), res)

		res, err = UnmarshalUint64(json.Number(" 1"))
		assert.EqualError(t, err, "strconv.ParseUint: parsing \" 1\": invalid syntax") //nolint:testifylint // An error assertion makes more sense.
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
