package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestIntrospection(t *testing.T) {
	rc := &graphql.OperationContext{
		DisableIntrospection: true,
	}
	require.NoError(t, Introspection{}.MutateOperationContext(context.Background(), rc))
	require.False(t, rc.DisableIntrospection)
}
