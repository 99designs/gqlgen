package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComplexityLimit(t *testing.T) {
	rc := testMiddleware(ComplexityLimitFunc(func(ctx context.Context) int {
		return 10
	}))

	require.True(t, rc.InvokedNext)
	require.Equal(t, 10, rc.ComplexityLimit)
}

func TestComplexityLimitFunc(t *testing.T) {
	rc := testMiddleware(ComplexityLimitFunc(func(ctx context.Context) int {
		return 22
	}))

	require.True(t, rc.InvokedNext)
	require.Equal(t, 22, rc.ComplexityLimit)
}
