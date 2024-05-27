package extension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

func TestIntrospection(t *testing.T) {
	rc := &graphql.OperationContext{
		DisableIntrospection: true,
	}
	err := Introspection{}.MutateOperationContext(context.Background(), rc)
	require.Equal(t, (*gqlerror.Error)(nil), err)
	require.False(t, rc.DisableIntrospection)
}
