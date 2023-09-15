package graphql

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUUID(t *testing.T) {
	t.Run("Null Values", func(t *testing.T) {
		assert.Equal(t, "null", m2s(MarshalUUID(uuid.Nil)))
	})

	t.Run("Valid Values", func(t *testing.T) {

		var values = []struct {
			input    uuid.UUID
			expected string
		}{
			{uuid.MustParse("fd5343a9-0372-11ee-9fb2-0242ac160014"), "\"fd5343a9-0372-11ee-9fb2-0242ac160014\""},
		}
		for _, v := range values {
			assert.Equal(t, v.expected, m2s(MarshalUUID(v.input)))
		}
	})
}

func TestUnmarshalUUID(t *testing.T) {
	t.Run("Invalid Non-String Values", func(t *testing.T) {
		var values = []interface{}{123, 1.2345678901, 1.2e+20, 1.2e-20, true, false, nil}
		for _, v := range values {
			result, err := UnmarshalUUID(v)
			assert.Equal(t, uuid.Nil, result)
			assert.ErrorContains(t, err, "is not a uuid")
		}
	})

	t.Run("Invalid String Values", func(t *testing.T) {
		var values = []struct {
			input    string
			expected string
		}{
			{"X50e8400-e29b-41d4-a716-446655440000", "invalid UUID format"},
			{"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "invalid UUID format"},
			{"F50e8400-e29b-41d4-a716-44665544000", "invalid UUID length: 35"},
			{"aaa", "invalid UUID length: 3"},
			{"", "invalid UUID length: 0"},
		}
		for _, v := range values {
			result, err := UnmarshalUUID(v.input)
			assert.Equal(t, uuid.Nil, result)
			assert.ErrorContains(t, err, v.expected)
		}
	})
}
