package graphql

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeferred(t *testing.T) {
	result := Defer(func() Marshaler {
		time.Sleep(10 * time.Millisecond)
		return Null
	})

	var b bytes.Buffer
	result.MarshalGQL(&b)
	require.Equal(t, "null", b.String())
}
