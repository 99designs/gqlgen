package graphql

import (
	"testing"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/assert"
)

func TestCoerceList(t *testing.T) {
	mapInput := map[string]any{
		"test": "value",
		"nested": map[string]any{
			"nested": true,
		},
	}

	jsonNumber := json.Number("12")

	assert.Equal(t, []any{"test", "values"}, CoerceList([]any{"test", "values"}))
	assert.Equal(t, []any{"test"}, CoerceList("test"))
	assert.Equal(t, []any{"test"}, CoerceList([]string{"test"}))
	assert.Equal(t, []any{3}, CoerceList([]int{3}))
	assert.Equal(t, []any{3}, CoerceList(3))
	assert.Equal(t, []any{int32(3)}, CoerceList([]int32{3}))
	assert.Equal(t, []any{int64(2)}, CoerceList([]int64{2}))
	assert.Equal(t, []any{float32(3.14)}, CoerceList([]float32{3.14}))
	assert.Equal(t, []any{3.14}, CoerceList([]float64{3.14}))
	assert.Equal(t, []any{jsonNumber}, CoerceList([]json.Number{jsonNumber}))
	assert.Equal(t, []any{jsonNumber}, CoerceList(jsonNumber))
	assert.Equal(t, []any{true}, CoerceList([]bool{true}))
	assert.Equal(t, []any{mapInput}, CoerceList(mapInput))
	assert.Equal(t, []any{mapInput}, CoerceList([]any{mapInput}))
	assert.Equal(t, []any{mapInput}, CoerceList([]map[string]any{mapInput}))
	assert.Empty(t, CoerceList(nil))
}
