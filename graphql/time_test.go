package graphql

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTime(t *testing.T) {
	t.Run("symmetry", func(t *testing.T) {
		initialTime := time.Now()
		buf := bytes.NewBuffer([]byte{})
		MarshalTime(initialTime).MarshalGQL(buf)

		str, err := strconv.Unquote(buf.String())
		require.NoError(t, err)
		newTime, err := UnmarshalTime(str)
		require.NoError(t, err)

		require.True(t, initialTime.Equal(newTime), "expected times %v and %v to equal", initialTime, newTime)
	})
}
