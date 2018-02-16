package graphql

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func UnmarshalComplexArg(result interface{}, data interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName:     "graphql",
		ErrorUnused: true,
		Result:      result,
		DecodeHook:  decodeHook,
	})
	if err != nil {
		panic(err)
	}

	return decoder.Decode(data)
}

func decodeHook(sourceType reflect.Type, destType reflect.Type, value interface{}) (interface{}, error) {
	return value, nil
}
