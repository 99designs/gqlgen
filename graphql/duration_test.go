package graphql

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationMarshaling(t *testing.T) {
	t.Run("UnmarshalDuration", func(t *testing.T) {
		d, err := UnmarshalDuration("P2Y")
		require.NoError(t, err)

		assert.InEpsilon(t, float64(365*24*2), d.Hours(), 0.02)
	})
	t.Run("MarshalDuration", func(t *testing.T) {
		m := MarshalDuration(time.Hour * 365 * 24 * 2)

		buf := new(bytes.Buffer)
		m.MarshalGQL(buf)

		assert.Equal(t, "\"P2Y\"", buf.String())
	})
}
