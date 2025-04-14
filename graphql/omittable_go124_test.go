//go:build go1.24

package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmittable_MarshalJSONAFromGo124(t *testing.T) {
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
