package graphql

import (
	"context"
	"errors"
	"reflect"
)

const unmarshalInputCtx key = "unmarshal_input_context"

// BuildUnmarshalerMap returns a map of unmarshal functions of the ExecutableContext
// to use with the WithUnmarshalerMap function.
func BuildUnmarshalerMap(unmarshaler ...interface{}) map[reflect.Type]reflect.Value {
	maps := make(map[reflect.Type]reflect.Value)
	for _, v := range unmarshaler {
		ft := reflect.TypeOf(v)
		if ft.Kind() == reflect.Func {
			maps[ft.Out(0)] = reflect.ValueOf(v)
		}
	}

	return maps
}

func WithUnmarshalerMap(ctx context.Context, maps map[reflect.Type]reflect.Value) context.Context {
	return context.WithValue(ctx, unmarshalInputCtx, maps)
}

// UnmarshalInputFromContext allows unmarshaling input object from a context.
func UnmarshalInputFromContext(ctx context.Context, raw, inputObj interface{}) error {
	m, ok := ctx.Value(unmarshalInputCtx).(map[reflect.Type]reflect.Value)
	if m == nil || !ok {
		return nil
	}

	out := reflect.ValueOf(inputObj)
	if out.Kind() != reflect.Ptr {
		return errors.New("input must be a pointer")
	}
	if fn, ok := m[out.Elem().Type()]; ok {
		res := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(raw),
		})
		if v := res[1].Interface(); v != nil {
			return v.(error)
		}

		out.Elem().Set(res[0])
		return nil
	}

	return nil
}
