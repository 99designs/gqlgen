package graphql

import (
	"fmt"
	"reflect"
	"time"
)

var Time = &ScalarConfig{
	ReflectType: reflect.TypeOf(time.Time{}),
	CoerceInput: func(input interface{}) (interface{}, error) {
		switch input := input.(type) {
		case time.Time:
			return input, nil
		case string:
			t, err := time.Parse(time.RFC3339, input)
			return t, err
		case int:
			return time.Unix(int64(input), 0), nil
		default:
			return nil, fmt.Errorf("wrong type")
		}
	},
}
