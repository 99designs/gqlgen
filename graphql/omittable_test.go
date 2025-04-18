package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
