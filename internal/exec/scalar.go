package exec

import (
	"fmt"
	"math"
	"reflect"

	"github.com/neelance/graphql-go/internal/schema"
)

type scalar struct {
	name        string
	reflectType reflect.Type
	coerceInput func(input interface{}) (interface{}, error)
}

func (*scalar) Kind() string       { return "SCALAR" }
func (t *scalar) TypeName() string { return t.name }

var intScalar = &scalar{
	name:        "Int",
	reflectType: reflect.TypeOf(int32(0)),
	coerceInput: func(input interface{}) (interface{}, error) {
		switch input := input.(type) {
		case int32:
			return input, nil
		case int:
			if input < math.MinInt32 || input > math.MaxInt32 {
				return nil, fmt.Errorf("not a 32-bit integer")
			}
			return int32(input), nil
		case float64:
			coerced := int32(input)
			if input < math.MinInt32 || input > math.MaxInt32 || float64(coerced) != input {
				return nil, fmt.Errorf("not a 32-bit integer")
			}
			return coerced, nil
		default:
			return nil, fmt.Errorf("wrong type")
		}
	},
}
var floatScalar = &scalar{
	name:        "Float",
	reflectType: reflect.TypeOf(float64(0)),
	coerceInput: func(input interface{}) (interface{}, error) {
		return input, nil // TODO
	},
}
var stringScalar = &scalar{
	name:        "String",
	reflectType: reflect.TypeOf(""),
	coerceInput: func(input interface{}) (interface{}, error) {
		return input, nil // TODO
	},
}
var booleanScalar = &scalar{
	name:        "Boolean",
	reflectType: reflect.TypeOf(true),
	coerceInput: func(input interface{}) (interface{}, error) {
		return input, nil // TODO
	},
}

var builtinScalars = []*scalar{
	intScalar,
	floatScalar,
	stringScalar,
	booleanScalar,
	// ID is defined in package "graphql"
}

func AddBuiltinScalars(s *schema.Schema) {
	for _, scalar := range builtinScalars {
		s.Types[scalar.name] = scalar
	}
}

func AddCustomScalar(s *schema.Schema, name string, reflectType reflect.Type, coerceInput func(input interface{}) (interface{}, error)) {
	s.Types[name] = &scalar{
		name:        name,
		reflectType: reflectType,
		coerceInput: coerceInput,
	}
}
