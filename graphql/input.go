package graphql

import (
	"context"
	"errors"
	"reflect"
)

const unmarshalInputCtx key = "unmarshal_input_context"

func BuildMap(unmarshaler ...interface{}) map[reflect.Type]reflect.Value {
	maps := make(map[reflect.Type]reflect.Value)
	for _, v := range unmarshaler {
		ft := reflect.TypeOf(v)
		if ft.Kind() != reflect.Func {
			panic("unmarshaler must be a function")
		}

		maps[ft.Out(0)] = reflect.ValueOf(v)
	}

	return maps
}

func WithUnmarshalerMap(ctx context.Context, maps map[reflect.Type]reflect.Value) context.Context {
	return context.WithValue(ctx, unmarshalInputCtx, maps)
}

func UnmarshalInputFromContext(ctx context.Context, data, input interface{}) error {
	m, ok := ctx.Value(unmarshalInputCtx).(map[reflect.Type]reflect.Value)
	if m == nil || !ok {
		return nil
	}

	out := reflect.ValueOf(input)
	if out.Kind() != reflect.Ptr {
		return errors.New("input must be a pointer")
	}
	if fn, ok := m[out.Elem().Type()]; ok {
		res := fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(data),
		})
		if v := res[1].Interface(); v != nil {
			return v.(error)
		}

		out.Elem().Set(res[0])
		return nil
	}

	return nil
}
