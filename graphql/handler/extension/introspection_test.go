package extension

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/require"
)

func TestIntrospection(t *testing.T) {
	rc := &graphql.RequestContext{
		DisableIntrospection: true,
	}
	require.Nil(t, Introspection{}.MutateRequestContext(context.Background(), rc))
	require.Equal(t, false, rc.DisableIntrospection)
}
