package nopanichandler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql/executor"
)

func TestSubscription(t *testing.T) {
	resolvers := &Stub{}
	resolvers.SubscriptionResolver.Count = func(ctx context.Context) (<-chan int, error) {
		ch := make(chan int, 2)
		ch <- 1
		ch <- 2
		close(ch)
		return ch, nil
	}
	resolvers.SubscriptionResolver.Failing = func(ctx context.Context) (<-chan int, error) {
		return nil, errors.New("failing stream")
	}
	resolvers.SubscriptionResolver.Panicking = func(ctx context.Context) (<-chan *int, error) {
		panic("subscription panicked")
	}
	exec := executor.New(NewExecutableSchema(Config{Resolvers: resolvers}))

	for _, tc := range []struct{ query, want string }{
		{`subscription { count }`, `[{"data":{"count":1}},{"data":{"count":2}}]`},
		{
			`subscription { failing }`,
			`[{"errors":[{"message":"failing stream","path":["failing"]}],"data":null}]`,
		},
		{`subscription { panicking }`, `"panic: subscription panicked"`},
		{
			`subscription { count @skip(if: true) }`,
			`[{"errors":[{"message":"must subscribe to exactly one stream"}],"data":null}]`,
		},
	} {
		require.JSONEq(t, tc.want, exectest.Subscribe(t, exec, tc.query), tc.query)
	}
}
