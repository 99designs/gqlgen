package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFieldInputContext(t *testing.T) {
	require.Nil(t, GetFieldContext(context.Background()))

	rc := &FieldInputContext{}
	require.Equal(t, rc, GetFieldInputContext(WithFieldInputContext(context.Background(), rc)))
}
