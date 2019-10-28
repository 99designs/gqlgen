package middleware

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
)

func TestTracer(t *testing.T) {
	tracer := &graphql.NopTracer{}
	rc := testMiddleware(Tracer(tracer))

	require.True(t, rc.InvokedNext)
	require.Equal(t, tracer, rc.ResultContext.Tracer)
	require.NotNil(t, tracer, rc.ResultContext.RequestMiddleware)
}
