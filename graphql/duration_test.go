package graphql

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDurationMarshaling(t *testing.T) {
	t.Run("UnmarshalDuration", func(t *testing.T) {
		d, err := UnmarshalDuration("P2Y")
		assert.NoError(t, err)

		assert.Equal(t, float64(365*24*2), d.Hours())
	})
	t.Run("MarshalDuration", func(t *testing.T) {
		m := MarshalDuration(time.Hour * 365 * 24 * 2)

		buf := new(bytes.Buffer)
		m.MarshalGQL(buf)

		assert.Equal(t, "\"P2Y\"", buf.String())
	})
}
