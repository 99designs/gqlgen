package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"golang.org/x/exp/constraints"
)

func MarshalUint(i uint) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint(v interface{}) (uint, error) {
	switch v := v.(type) {
	case string:
		u64, err := strconv.ParseUint(v, 10, 64)
		return uint(u64), err
	case int:
		return safeUintCast[int, uint](v)
	case int64:
		return safeUintCast[int64, uint](v)
	case json.Number:
		u64, err := strconv.ParseUint(string(v), 10, 64)
		return uint(u64), err
	default:
		return 0, fmt.Errorf("%T is not an uint", v)
	}
}

func MarshalUint64(i uint64) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(i, 10))
	})
}

func UnmarshalUint64(v interface{}) (uint64, error) {
	switch v := v.(type) {
	case string:
		return strconv.ParseUint(v, 10, 64)
	case int:
		return safeUintCast[int, uint64](v)
	case int64:
		return safeUintCast[int64, uint64](v)
	case json.Number:
		return strconv.ParseUint(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("%T is not an uint", v)
	}
}

func MarshalUint32(i uint32) Marshaler {
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUint32(v interface{}) (uint32, error) {
	switch v := v.(type) {
	case string:
		iv, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, err
		}
		return uint32(iv), nil
	case int:
		return safeUintCast[int, uint32](v)
	case int64:
		return safeUintCast[int64, uint32](v)
	case json.Number:
		iv, err := strconv.ParseUint(string(v), 10, 32)
		if err != nil {
			return 0, err
		}
		return uint32(iv), nil
	default:
		return 0, fmt.Errorf("%T is not an uint", v)
	}
}

func safeUintCast[F constraints.Signed, T constraints.Unsigned](f F) (T, error) {
	if f < 0 {
		return 0, fmt.Errorf("cannot cast %d to uint", f)
	}
	return T(f), nil
}
