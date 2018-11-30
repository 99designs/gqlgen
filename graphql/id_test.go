package graphql

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCases struct {
	expect string
	value  interface{}
	err    error
}

func TestUnmarshalID(t *testing.T) {
	cases := map[string]testCases{
		"string": testCases{
			value:  "best id",
			expect: "best id",
			err:    nil,
		},
		"json.Number": testCases{
			value:  json.Number("json 42"),
			expect: "json 42",
			err:    nil,
		},
		"int": testCases{
			value:  int(42),
			expect: "42",
			err:    nil,
		},
		"max int": testCases{
			value:  math.MaxInt32,
			expect: "2147483647",
			err:    nil,
		},
		"negative int": testCases{
			value:  int(-42),
			expect: "-42",
			err:    nil,
		},
		"int32": testCases{
			value:  int32(42),
			expect: "42",
			err:    nil,
		},
		"negative int32": testCases{
			value:  int32(-42),
			expect: "-42",
			err:    nil,
		},
		"int64": testCases{
			value:  int64(42),
			expect: "42",
			err:    nil,
		},
		"max int64": testCases{
			value:  math.MaxInt64,
			expect: "9223372036854775807",
			err:    nil,
		},
		"negative int64": testCases{
			value:  int64(-42),
			expect: "-42",
			err:    nil,
		},
		"uint64": testCases{
			value:  uint64(42),
			expect: "42",
			err:    nil,
		},
		"max uint64": testCases{
			value:  uint64(math.MaxUint64),
			expect: "18446744073709551615",
			err:    nil,
		},
		"uint": testCases{
			value:  uint(42),
			expect: "42",
			err:    nil,
		},
		"max uint": testCases{
			value:  uint(math.MaxUint32),
			expect: "4294967295",
			err:    nil,
		},
		"uint8": testCases{
			value:  uint8(42),
			expect: "42",
			err:    nil,
		},
		"max uint8": testCases{
			value:  math.MaxUint8,
			expect: "255",
			err:    nil,
		},
		"uint16": testCases{
			value:  uint16(42),
			expect: "42",
			err:    nil,
		},
		"max uint16": testCases{
			value:  math.MaxUint16,
			expect: "65535",
			err:    nil,
		},
		"uint32": testCases{
			value:  uint32(42),
			expect: "42",
			err:    nil,
		},
		"max uint32": testCases{
			value:  math.MaxUint32,
			expect: "4294967295",
			err:    nil,
		},
		"float32": testCases{
			value:  float32(42.0),
			expect: "42.000000",
			err:    nil,
		},
		"MaxFloat32": testCases{
			value:  math.MaxFloat32,
			expect: "340282346638528859811704183484516925440.000000",
			err:    nil,
		},
		"Pi": testCases{
			value:  math.Pi,
			expect: "3.141593",
			err:    nil,
		},
		"negative float32": testCases{
			value:  float32(-42.0),
			expect: "-42.000000",
			err:    nil,
		},
		"float64": testCases{
			value:  float64(42.000001),
			expect: "42.000001",
			err:    nil,
		},
		"negative float64": testCases{
			value:  float64(-42.01),
			expect: "-42.010000",
			err:    nil,
		},
		"true": testCases{
			value:  true,
			expect: "true",
			err:    nil,
		},
		"false": testCases{
			value:  false,
			expect: "false",
			err:    nil,
		},
		"nil": testCases{
			value:  nil,
			expect: "null",
			err:    nil,
		},
		"error": testCases{
			value:  struct{}{},
			expect: "",
			err:    fmt.Errorf("%T is not a string", struct{}{}),
		},
	}
	for name, cas := range cases {
		res, err := UnmarshalID(cas.value)
		require.Equal(t, cas.err, err, name)
		require.Equal(t, cas.expect, res, name)
	}
}
