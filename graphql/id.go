package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func MarshalID(s string) Marshaler {
	return MarshalString(s)
}

func UnmarshalID(v interface{}) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case json.Number:
		return string(v), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		if v {
			return "true", nil
		} else {
			return "false", nil
		}
	case nil:
		return "null", nil
	default:
		return "", fmt.Errorf("%T is not a string", v)
	}
}

func MarshalIntID(i int) Marshaler {
	return WriterFunc(func(w io.Writer) {
		writeQuotedString(w, strconv.Itoa(i))
	})
}

func UnmarshalIntID(v interface{}) (int, error) {
	switch v := v.(type) {
	case string:
		return strconv.Atoi(v)
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case json.Number:
		return strconv.Atoi(string(v))
	default:
		return 0, fmt.Errorf("%T is not an int", v)
	}
}

func MarshalUintID(i uint) Marshaler {
	return WriterFunc(func(w io.Writer) {
		writeQuotedString(w, strconv.FormatUint(uint64(i), 10))
	})
}

func UnmarshalUintID(v interface{}) (uint, error) {
	switch v := v.(type) {
	case string:
		result, err := strconv.ParseUint(v, 10, 64)
		return uint(result), err
	case int:
		return uint(v), nil
	case int64:
		return uint(v), nil
	case int32:
		return uint(v), nil
	case uint32:
		return uint(v), nil
	case uint64:
		return uint(v), nil
	case json.Number:
		result, err := strconv.ParseUint(string(v), 10, 64)
		return uint(result), err
	default:
		return 0, fmt.Errorf("%T is not an uint", v)
	}
}
