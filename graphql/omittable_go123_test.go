//go:build !go1.24

package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmittable_MarshalJSONBeforeGo124(t *testing.T) {
	testCases := []struct {
		name         string
		input        any
		expectedJSON string
	}{
		{
			name: "simple string omitempty IsZero=true",
			input: struct {
				Value Omittable[string] `json:",omitempty"`
			}{},
			expectedJSON: `{"Value":""}`,
		},
		{
			name: "string pointer omitempty IsZero=true",
			input: struct {
				Value Omittable[*string] `json:",omitempty"`
			}{},
			expectedJSON: `{"Value":null}`,
		},
		{
			name: "omitted integer omitempty IsZero=true",
			input: struct {
				Value Omittable[int] `json:",omitempty"`
			}{},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omittable omittable omitempty IsZero=true", //nolint:dupword
			input: struct {
				Value Omittable[Omittable[uint64]] `json:",omitempty"`
			}{},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omittable struct Value omitempty IsZero=true",
			input: struct {
				Value Omittable[struct {
					Inner string
				}] `json:",omitempty"`
			}{},
			expectedJSON: `{"Value":{"Inner":""}}`,
		},
		{
			name: "omittable struct Inner omitempty IsZero=true",
			input: struct {
				Value Omittable[struct {
					Inner Omittable[string] `json:",omitempty"`
				}]
			}{},
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
