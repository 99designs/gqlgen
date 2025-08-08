package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

func MarshalUint(i uint) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint(v any) (uint, error) {
	return interfaceToUnsignedNumber[uint](v)
}

func MarshalUint8(i uint8) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint8(v any) (uint8, error) {
	return interfaceToUnsignedNumber[uint8](v)
}

func MarshalUint16(i uint16) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint16(v any) (uint16, error) {
	return interfaceToUnsignedNumber[uint16](v)
}

func MarshalUint32(i uint32) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint32(v any) (uint32, error) {
	return interfaceToUnsignedNumber[uint32](v)
}

func MarshalUint64(i uint64) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(i, 10))
	})
}

func UnmarshalUint64(v any) (uint64, error) {
	return interfaceToUnsignedNumber[uint64](v)
}

func interfaceToUnsignedNumber[N number](v any) (N, error) {
	switch v := v.(type) {
	case int:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(int64(v), 10))
		}
		return safeCastUnsignedNumber[N](uint64(v))
	case int8:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(int64(v), 10))
		}
		return safeCastUnsignedNumber[N](uint64(v))
	case int16:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(int64(v), 10))
		}
		return safeCastUnsignedNumber[N](uint64(v))
	case int32:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(int64(v), 10))
		}
		return safeCastUnsignedNumber[N](uint64(v))
	case int64:
		if v < 0 {
			return 0, newUintSignError(strconv.FormatInt(int64(v), 10))
		}
		return safeCastUnsignedNumber[N](uint64(v))
	case uint:
		return safeCastUnsignedNumber[N](uint64(v))
	case uint8:
		return safeCastUnsignedNumber[N](uint64(v))
	case uint16:
		return safeCastUnsignedNumber[N](uint64(v))
	case uint32:
		return safeCastUnsignedNumber[N](uint64(v))
	case uint64:
		return safeCastUnsignedNumber[N](uint64(v))
	case string:
		uv, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			var strconvErr *strconv.NumError
			if errors.As(err, &strconvErr) && isSignedInteger(v) {
				return 0, newUintSignError(v)
			}
			return 0, err
		}
		return safeCastUnsignedNumber[N](uv)
	case json.Number:
		uv, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			var strconvErr *strconv.NumError
			if errors.As(err, &strconvErr) && isSignedInteger(string(v)) {
				return 0, newUintSignError(string(v))
			}
			return 0, err
		}
		return safeCastUnsignedNumber[N](uv)
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not an %T", v, N(0))
	}
}

type UintSignError struct {
	*IntegerError
}

func newUintSignError(v string) *UintSignError {
	return &UintSignError{
		IntegerError: &IntegerError{
			Message: fmt.Sprintf("%v is an invalid unsigned integer: includes sign", v),
		},
	}
}

func (e *UintSignError) Unwrap() error {
	return e.IntegerError
}

// safeCastUnsignedNumber converts an uint64 to a number of type N.
func safeCastUnsignedNumber[N number](val uint64) (N, error) {
	bitsize := fmt.Sprintf("%T", N(0))
	switch bitsize {
	case "int8", "uint8":
		if val > math.MaxUint8 {
			return 0, newNumberOverflowError[uint64](val, 8)
		}
		return N(val), nil
	case "int16", "uint16":
		if val > math.MaxUint16 {
			return 0, newNumberOverflowError[uint64](val, 16)
		}
		return N(val), nil
	case "int32", "uint32":
		if val > math.MaxUint32 {
			return 0, newNumberOverflowError[uint64](val, 32)
		}
		return N(val), nil
	case "int64", "int", "uint64", "uint":
		return N(val), nil
	default:
		return 0, fmt.Errorf("invalid bitsize %s", bitsize)
	}
}

func isSignedInteger(v string) bool {
	if v == "" {
		return false
	}
	if v[0] != '-' && v[0] != '+' {
		return false
	}
	if _, err := strconv.ParseUint(v[1:], 10, 64); err == nil {
		return true
	}
	return false
}
