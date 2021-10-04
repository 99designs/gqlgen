package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func MarshalMap(val map[string]interface{}) Marshaler {
	return WriterFunc(func(w io.Writer) {
		err := json.NewEncoder(w).Encode(val)
		if err != nil {
			panic(err)
		}
	})
}

func UnmarshalMap(v interface{}) (map[string]interface{}, error) {
	unmarshalNumber(v)

	if m, ok := v.(map[string]interface{}); ok {
		return m, nil
	}

	return nil, fmt.Errorf("%T is not a map", v)
}

func unmarshalNumber(v interface{}) {
	if n, ok := v.(json.Number); ok {
		v, _ = strconv.Atoi(string(n))
	}

	switch v := v.(type) {
	case []interface{}:
		for _, v := range v {
			unmarshalNumber(v)
		}
	case map[string]interface{}:
		for _, v := range v {
			unmarshalNumber(v)
		}
	}
}
