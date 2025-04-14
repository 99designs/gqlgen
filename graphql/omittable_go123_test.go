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
			name: "simple string omitzero IsZero=true",
			input: struct {
				Value Omittable[string] `json:",omitzero"`
			}{},
			expectedJSON: `{"Value":""}`,
		},
		{
			name: "string pointer omitzero IsZero=true",
			input: struct {
				Value Omittable[*string] `json:",omitzero"`
			}{},
			expectedJSON: `{"Value":null}`,
		},
		{
			name: "omitted integer omitzero IsZero=true",
			input: struct {
				Value Omittable[int] `json:",omitzero"`
			}{},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omittable omittable omitzero IsZero=true", //nolint:dupword
			input: struct {
				Value Omittable[Omittable[uint64]] `json:",omitzero"`
			}{},
			expectedJSON: `{"Value":0}`,
		},
		{
			name: "omittable struct Value omitzero IsZero=true",
			input: struct {
				Value Omittable[struct {
					Inner string
				}] `json:",omitzero"`
			}{},
			expectedJSON: `{"Value":{"Inner":""}}`,
		},
		{
			name: "omittable struct Inner omitzero IsZero=true",
			input: struct {
				Value Omittable[struct {
					Inner Omittable[string] `json:",omitzero"`
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
