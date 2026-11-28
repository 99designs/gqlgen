package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPanicInResolver(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Ping = func(ctx context.Context) (string, error) {
		return "pong", nil
	}
	resolvers.QueryResolver.Panicking = func(ctx context.Context) (*string, error) {
		panic("resolver panicked")
	}
	resolvers.QueryResolver.PanickingNonNull = func(ctx context.Context) (string, error) {
		panic("resolver panicked")
	}
	resolvers.MutationResolver.Panicking = func(ctx context.Context) (*string, error) {
		panic("mutation panicked")
	}
	srv := newServer(resolvers, DirectiveRoot{})

	got := post(t, srv, `{ ping panicking }`)
	require.JSONEq(t, `{"data":{"ping":"pong","panicking":null},"errors":[
		{"message":"internal system error","path":["panicking"]}]}`, got)

	got = post(t, srv, `{ ping panickingNonNull }`)
	require.JSONEq(t, `{"data":null,"errors":[
		{"message":"internal system error","path":["panickingNonNull"]}]}`, got)

	got = post(t, srv, `mutation { panicking }`)
	require.JSONEq(t, `{"data":{"panicking":null},"errors":[
		{"message":"internal system error","path":["panicking"]}]}`, got)
}
