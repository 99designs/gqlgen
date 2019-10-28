package middleware

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntrospection(t *testing.T) {
	rc := testMiddleware(Introspection(), graphql.RequestContext{
		DisableIntrospection: true,
	})

	require.True(t, rc.InvokedNext)
	// cant test for function equality in go, so testing the return type instead
	assert.False(t, rc.ResultContext.DisableIntrospection)
}
