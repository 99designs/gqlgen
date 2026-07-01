package fedruntime

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestSplitEntityBatchErrors(t *testing.T) {
	t.Parallel()

	errA := errors.New("a")
	errB := errors.New("b")

	t.Run("nil error yields no per-index and no fatal", func(t *testing.T) {
		t.Parallel()
		perIndex, fatal := SplitEntityBatchErrors(nil)
		assert.Nil(t, perIndex)
		require.NoError(t, fatal)
	})

	t.Run("plain error is fatal for the whole batch", func(t *testing.T) {
		t.Parallel()
		perIndex, fatal := SplitEntityBatchErrors(errA)
		assert.Nil(t, perIndex)
		assert.Equal(t, errA, fatal)
	})

	t.Run("BatchErrorList becomes per-index with no fatal", func(t *testing.T) {
		t.Parallel()
		perIndex, fatal := SplitEntityBatchErrors(graphql.BatchErrorList{errA, nil, errB})
		require.NoError(t, fatal)
		require.Len(t, perIndex, 3)
		assert.Equal(t, errA, perIndex[0])
		require.NoError(t, perIndex[1])
		assert.Equal(t, errB, perIndex[2])
	})

	t.Run("wrapped BatchErrors is still detected", func(t *testing.T) {
		t.Parallel()
		wrapped := fmt.Errorf("batch failed: %w", graphql.BatchErrorList{errA})
		perIndex, fatal := SplitEntityBatchErrors(wrapped)
		require.NoError(t, fatal)
		require.Len(t, perIndex, 1)
		assert.Equal(t, errA, perIndex[0])
	})
}
