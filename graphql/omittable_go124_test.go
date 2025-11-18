//go:build go1.24

package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmittableIsZeroTrue_MarshalJSONGo124(t *testing.T) {
	s := "test"
	testCases := []struct {
		name         string
		input        any
		expectedJSON string
	}{
		{
			name: "simple string omitzero IsZero=true",
			input: struct {
				Value Omittable[string] `json:",omitzero"`
			}{},
			expectedJSON: `{}`,
		},
		{
			name:         "string pointer",
			input:        struct{ Value Omittable[*string] }{Value: OmittableOf(&s)},
			expectedJSON: `{"Value":"test"}`,
		},
		{
			name: "string pointer omitzero IsZero=true",
			input: struct {
				Value Omittable[*string] `json:",omitzero"`
			}{},
			expectedJSON: `{}`,
		},
		{
			name:         "omitted integer",
			input:        struct{ Value Omittable[int] }{},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omitted integer omitzero IsZero=true",
			input: struct {
				Value Omittable[int] `json:",omitzero"`
			}{},
			expectedJSON: `{}`,
		},
		{
			name: "omittable omittable omitzero IsZero=true", //nolint:dupword
			input: struct {
				Value Omittable[Omittable[uint64]] `json:",omitzero"`
			}{},
			expectedJSON: `{}`,
		},
		{
			name: "omittable struct Value omitzero IsZero=true",
			input: struct {
				Value Omittable[struct {
					Inner string
				}] `json:",omitzero"`
			}{},
			expectedJSON: `{}`,
		},
		{
			name: "omittable struct Inner omitzero IsZero=true",
			input: struct {
				Value Omittable[struct {
					Inner Omittable[string] `json:",omitzero"`
				}]
			}{},
			expectedJSON: `{"Value":{}}`,
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

func TestOmittableIsZeroFalse_MarshalJSONGo124(t *testing.T) {
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
			name: "omittable omittable", //nolint:dupword
			input: struct{ Value Omittable[Omittable[uint64]] }{
				Value: OmittableOf(OmittableOf(uint64(42))),
			},
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
