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
			expectedJSON: `{"Value":"simple string"}`,
		},
		{
			name: "simple string omitzero IsZero=false",
			input: struct {
				Value Omittable[string] `json:",omitzero"`
			}{
				Value: OmittableOf(""),
			},
			expectedJSON: `{"Value":""}`,
		},
		{
			name:         "string pointer",
			input:        struct{ Value Omittable[*string] }{Value: OmittableOf(&s)},
			expectedJSON: `{"Value":"test"}`,
		},
		{
			name: "string pointer omitzero IsZero=false",
			input: struct {
				Value Omittable[*string] `json:",omitzero"`
			}{
				Value: OmittableOf[*string](nil),
			},
			expectedJSON: `{"Value":null}`,
		},
		{
			name:         "omitted integer",
			input:        struct{ Value Omittable[int] }{},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omitted integer omitzero IsZero=false",
			input: struct {
				Value Omittable[int] `json:",omitzero"`
			}{
				Value: OmittableOf(0),
			},
			expectedJSON: `{"Value":0}`,
		},
		{
			name:         "omittable omittable", //nolint:dupword
			input:        struct{ Value Omittable[Omittable[uint64]] }{Value: OmittableOf(OmittableOf(uint64(42)))},
			expectedJSON: `{"Value":42}`,
		},
		{
			name: "omittable omittable omitzero IsZero=false", //nolint:dupword
			input: struct {
				Value Omittable[Omittable[uint64]] `json:",omitzero"`
			}{
				Value: OmittableOf(OmittableOf(uint64(0))),
			},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omittable struct",
			input: struct {
				Value Omittable[struct{ Inner string }]
			}{Value: OmittableOf(struct{ Inner string }{})},
			expectedJSON: `{"Value":{"Inner":""}}`,
		},
		{
			name: "omittable struct Value omitzero IsZero=false",
			input: struct {
				Value Omittable[struct {
					Inner string
				}] `json:",omitzero"`
			}{
				Value: OmittableOf(struct {
					Inner string
				}{
					Inner: "",
				}),
			},
			expectedJSON: `{"Value":{"Inner":""}}`,
		},
		{
			name: "omittable struct Inner omitzero IsZero=false",
			input: struct {
				Value Omittable[struct {
					Inner Omittable[string] `json:",omitzero"`
				}]
			}{
				Value: OmittableOf(struct {
					Inner Omittable[string] `json:",omitzero"`
				}{
					Inner: OmittableOf(""),
				}),
			},
			expectedJSON: `{"Value":{"Inner":""}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedJSON, string(data))
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
