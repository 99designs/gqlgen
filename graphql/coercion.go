package graphql

import (
	"encoding/json"
)

// CoerceList applies coercion from a single value to a list.
func CoerceList(v any) []any {
	if v == nil {
		return nil
	}

	switch v := v.(type) {
	case []any:
		// already a slice no coercion required
		return v
	case []string:
		return toAnySlice(v)
	case []json.Number:
		return toAnySlice(v)
	case []bool:
		return toAnySlice(v)
	case []map[string]any:
		return toAnySlice(v)
	case []float64:
		return toAnySlice(v)
	case []float32:
		return toAnySlice(v)
	case []int:
		return toAnySlice(v)
	case []int32:
		return toAnySlice(v)
	case []int64:
		return toAnySlice(v)
	default:
		return []any{v}
	}
}

func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
