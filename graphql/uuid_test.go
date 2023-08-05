package graphql

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUUID(t *testing.T) {
	t.Run("Null Values", func(t *testing.T) {
		var input = []uuid.UUID{uuid.Nil, uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000")}
		for _, v := range input {
			assert.Equal(t, Null, MarshalUUID(v))
		}
	})

	t.Run("Valid Values", func(t *testing.T) {
		var generator = uuid.NewGen()
		var v1, _ = generator.NewV1()
		var v3 = generator.NewV3(uuid.FromStringOrNil("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), "gqlgen.com")
		var v4, _ = generator.NewV4()
		var v5 = generator.NewV5(uuid.FromStringOrNil("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), "gqlgen.com")
		var v6, _ = generator.NewV6()
		var v7, _ = generator.NewV7()
		var values = []struct {
			input    uuid.UUID
			expected string
		}{
			{v1, v1.String()},
			{v3, v3.String()},
			{v4, v4.String()},
			{v5, v5.String()},
			{v6, v6.String()},
			{v7, v7.String()},
		}
		for _, v := range values {
			assert.Equal(t, v.expected, m2s(MarshalUUID(v.input)))
		}
	})
}

func TestUnmarshalUUID(t *testing.T) {
	t.Run("Invalid Non-String Values", func(t *testing.T) {
		var values = []interface{}{123, 1.2345678901, 1.2e+20, 1.2e-20, true, false}
		for _, v := range values {
			result, err := UnmarshalUUID(v)
			assert.Equal(t, uuid.Nil, result)
			assert.ErrorContains(t, err, "input must be an RFC-4122 formatted string")
		}
	})

	t.Run("Invalid String Values", func(t *testing.T) {
		var values = []struct {
			input    string
			expected string
		}{
			{"x50e8400-e29b-41d4-a716-446655440000", "uuid: invalid UUID format"},
			{"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "uuid: invalid UUID format"},
			{"f50e8400-e29b-41d4-a716-44665544000", "uuid: incorrect UUID length 35 in string"},
			{"foo", "uuid: incorrect UUID length 3 in string"},
			{"", "uuid: incorrect UUID length 0 in string"},
		}
		for _, v := range values {
			result, err := UnmarshalUUID(v.input)
			assert.Equal(t, uuid.Nil, result)
			assert.ErrorContains(t, err, v.expected)
		}
	})

	t.Run("Valid Values", func(t *testing.T) {
		var generator = uuid.NewGen()
		var v1, _ = generator.NewV1()
		var v3 = generator.NewV3(uuid.FromStringOrNil("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), "gqlgen.com")
		var v4, _ = generator.NewV4()
		var v5 = generator.NewV5(uuid.FromStringOrNil("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), "gqlgen.com")
		var v6, _ = generator.NewV6()
		var v7, _ = generator.NewV7()
		var values = []struct {
			input    string
			expected uuid.UUID
		}{
			{v1.String(), v1},
			{v3.String(), v3},
			{v4.String(), v4},
			{v5.String(), v5},
			{v6.String(), v6},
			{v7.String(), v7},
		}
		for _, v := range values {
			result, err := UnmarshalUUID(v.input)
			assert.Equal(t, v.expected, result)
			assert.Nil(t, err)
		}
	})
}
