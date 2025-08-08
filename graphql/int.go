package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
)

func MarshalInt(i int) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatInt(int64(i), 10))
	})
}

func UnmarshalInt(v any) (int, error) {
	return interfaceToNumber[int](v)
}

func MarshalInt8(i int8) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatInt(int64(i), 10))
	})
}

func UnmarshalInt8(v any) (int8, error) {
	return interfaceToNumber[int8](v)
}

func MarshalInt16(i int16) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatInt(int64(i), 10))
	})
}

func UnmarshalInt16(v any) (int16, error) {
	return interfaceToNumber[int16](v)
}

func MarshalInt32(i int32) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatInt(int64(i), 10))
	})
}

func UnmarshalInt32(v any) (int32, error) {
	return interfaceToNumber[int32](v)
}

func MarshalInt64(i int64) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatInt(i, 10))
	})
}

func UnmarshalInt64(v any) (int64, error) {
	return interfaceToNumber[int64](v)
}

type number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64
}

func interfaceToNumber[N number](v any) (N, error) {
	switch v := v.(type) {
	case int:
		return safeCastNumber[N](int64(v))
	case int8:
		return safeCastNumber[N](int64(v))
	case int16:
		return safeCastNumber[N](int64(v))
	case int32:
		return safeCastNumber[N](int64(v))
	case int64:
		return safeCastNumber[N](int64(v))
	case uint:
		return safeCastNumber[N](int64(v))
	case uint8:
		return safeCastNumber[N](int64(v))
	case uint16:
		return safeCastNumber[N](int64(v))
	case uint32:
		return safeCastNumber[N](int64(v))
	case uint64:
		return safeCastNumber[N](int64(v))
	case string:
		iv, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return safeCastNumber[N](iv)
	case json.Number:
		iv, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return 0, err
		}
		return safeCastNumber[N](iv)
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("%T is not an %T", v, N(0))
	}
}

// IntegerError is an error type that allows users to identify errors associated
// with receiving an integer value that is not valid for the specific integer
// type designated by the API. IntegerErrors designate otherwise valid unsigned
// or signed 64-bit integers that are invalid in a specific context: they do not
// designate integers that overflow 64-bit versions of the current type.
type IntegerError struct {
	Message string
}

func (e IntegerError) Error() string {
	return e.Message
}

type NumberOverflowError struct {
	Value int64
	*IntegerError
}

func newNumberOverflowError(i int64, bitsize int) *NumberOverflowError {
	return &NumberOverflowError{
		Value: i,
		IntegerError: &IntegerError{
			Message: fmt.Sprintf("%d overflows signed %d-bit integer", i, bitsize),
		},
	}
}

func (e *NumberOverflowError) Unwrap() error {
	return e.IntegerError
}

// safeCastNumber converts an int64 to a number of type N.
func safeCastNumber[N number](val int64) (N, error) {
	bitsize := fmt.Sprintf("%T", N(0))
	switch bitsize {
	case "int8":
		if val > math.MaxInt8 || val < math.MinInt8 {
			return 0, newNumberOverflowError(val, 8)
		}
		return N(val), nil
	case "uint8":
		if val > math.MaxUint8 || val < 0 {
			return 0, newNumberOverflowError(val, 8)
		}
		return N(val), nil
	case "int16":
		if val > math.MaxInt16 || val < math.MinInt16 {
			return 0, newNumberOverflowError(val, 16)
		}
		return N(val), nil
	case "uint16":
		if val > math.MaxUint16 || val < 0 {
			return 0, newNumberOverflowError(val, 16)
		}
		return N(val), nil
	case "int32", "int":
		if val > math.MaxInt32 || val < math.MinInt32 {
			return 0, newNumberOverflowError(val, 32)
		}
		return N(val), nil
	case "uint32", "uint":
		if val > math.MaxUint32 || val < 0 {
			return 0, newNumberOverflowError(val, 32)
		}
		return N(val), nil
	case "int64":
		return N(val), nil
	case "uint64":
		if val < 0 {
			return 0, newNumberOverflowError(val, 64)
		}
		return N(val), nil
	default:
		return 0, fmt.Errorf("invalid bitsize %s", bitsize)
	}
}
