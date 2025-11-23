package graphql

import (
	"bytes"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimal(t *testing.T) {
	t.Run("UnmarshalDecimal", func(t *testing.T) {
		d, err := UnmarshalDecimal("0.0")
		require.NoError(t, err)
		i := d.Cmp(decimal.Zero)

		assert.Equal(t, i, 0)
	})
	t.Run("MarshalDecimal", func(t *testing.T) {
		m := MarshalDecimal(decimal.Zero)

		buf := new(bytes.Buffer)
		m.MarshalGQL(buf)

		assert.Equal(t, "\"0\"", buf.String())
	})
}
