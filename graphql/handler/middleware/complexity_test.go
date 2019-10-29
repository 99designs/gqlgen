package middleware

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
)

func TestComplexityLimit(t *testing.T) {
	rc := &graphql.RequestContext{}
	ComplexityLimit(10).MutateRequestContext(context.Background(), rc)
	require.Equal(t, 10, rc.ComplexityLimit)
}
