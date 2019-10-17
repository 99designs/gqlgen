package graphql

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalID(t *testing.T) {
	tests := []struct {
		Name        string
		Input       interface{}
		Expected    string
		ShouldError bool
	}{
		{
			Name:        "int64",
			Input:       int64(12),
			Expected:    "12",
			ShouldError: false,
		},
		{
			Name:     "int64 max",
			Input:    math.MaxInt64,
			Expected: "9223372036854775807",
		},
		{
			Name:     "int64 min",
			Input:    math.MinInt64,
			Expected: "-9223372036854775808",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			id, err := UnmarshalID(tt.Input)

			assert.Equal(t, tt.Expected, id)
			if tt.ShouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
