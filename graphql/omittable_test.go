package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmittable_MarshalJSON(t *testing.T) {
	s := "test"
	testCases := []struct {
		name         string
		input        any
		expectedJSON string
	}{
		{
			name:         "simple string",
			input:        struct{ Value Omittable[string] }{Value: OmittableOf("simple string")},
			expectedJSON: `{"Value": "simple string"}`,
		},
		{
			name:         "string pointer",
			input:        struct{ Value Omittable[*string] }{Value: OmittableOf(&s)},
			expectedJSON: `{"Value": "test"}`,
		},
		{
			name:         "omitted integer",
			input:        struct{ Value Omittable[int] }{},
			expectedJSON: `{"Value": null}`,
		},
		{
			name:         "omittable omittable",
			input:        struct{ Value Omittable[Omittable[uint64]] }{Value: OmittableOf(OmittableOf(uint64(42)))},
			expectedJSON: `{"Value": 42}`,
		},
		{
			name: "omittable struct",
			input: struct {
				Value Omittable[struct{ Inner string }]
			}{Value: OmittableOf(struct{ Inner string }{})},
			expectedJSON: `{"Value": {"Inner": ""}}`,
		},
		{
			name: "omitted struct",
			input: struct {
				Value Omittable[struct{ Inner string }]
			}{},
			expectedJSON: `{"Value": null}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.input)
			require.NoError(t, err)
			assert.JSONEq(t, tc.expectedJSON, string(data))
		})
	}
}

func TestOmittable_UnmarshalJSON(t *testing.T) {
	var s struct {
		String        Omittable[string]
		OmittedString Omittable[string]
		StringPointer Omittable[*string]
		NullInt       Omittable[int]
	}

	err := json.Unmarshal([]byte(`
	{
		"String": "simple string",
		"StringPointer": "string pointer",
		"NullInt": null
	}`), &s)

	require.NoError(t, err)
	assert.Equal(t, "simple string", s.String.Value())
	assert.True(t, s.String.IsSet())
	assert.False(t, s.OmittedString.IsSet())
	assert.True(t, s.StringPointer.IsSet())
	if assert.NotNil(t, s.StringPointer.Value()) {
		assert.EqualValues(t, "string pointer", *s.StringPointer.Value())
	}
	assert.True(t, s.NullInt.IsSet())
	assert.Zero(t, s.NullInt.Value())
}
