package singlefile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestIssue4053(t *testing.T) {
	t.Run("sending null input2 should not panic", func(t *testing.T) {
		resolver := &Stub{}
		resolver.MutationResolver.Issue4053 = func(ctx context.Context, input *Issue4053Input1) (bool, error) {
			require.NotNil(t, input, "input should not be nil")
			assert.Zero(t, input.Input2, "input2 should be zero value when null is passed")
			return true, nil
		}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
		srv.AddTransport(transport.POST{})
		srv.SetRecoverFunc(nil) // disable panic recovery to allow test to fail if a panic occurs
		c := client.New(srv)

		require.NotPanics(t, func() {
			var resp struct {
				Issue4053 bool
			}
			err := c.Post(`mutation { issue4053(input: { input2: null }) }`, &resp)
			assert.NoError(t, err)
			assert.True(t, resp.Issue4053)
		}, "should not panic when input2 is null")
	})

	t.Run("not sending input1 should yield nil Issue4053Input1", func(t *testing.T) {
		resolver := &Stub{}
		resolver.MutationResolver.Issue4053 = func(ctx context.Context, input *Issue4053Input1) (bool, error) {
			require.Nil(t, input, "input should be nil when not sent")
			return true, nil
		}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
		srv.AddTransport(transport.POST{})
		srv.SetRecoverFunc(nil) // disable panic recovery to allow test to fail if a panic occurs
		c := client.New(srv)

		require.NotPanics(t, func() {
			var resp struct {
				Issue4053 bool
			}
			err := c.Post(`mutation { issue4053 }`, &resp)
			assert.NoError(t, err)
			assert.True(t, resp.Issue4053)
		}, "should not panic when input2 is null")
	})

	t.Run("sending empty input1 should yield zero value of Issue4053Input2", func(t *testing.T) {
		resolver := &Stub{}
		resolver.MutationResolver.Issue4053 = func(ctx context.Context, input *Issue4053Input1) (bool, error) {
			require.NotNil(t, input, "input should not be nil")
			assert.Zero(t, input.Input2, "input2 should be zero value when empty input is passed")
			return true, nil
		}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
		srv.AddTransport(transport.POST{})
		srv.SetRecoverFunc(nil) // disable panic recovery to allow test to fail if a panic occurs
		c := client.New(srv)

		require.NotPanics(t, func() {
			var resp struct {
				Issue4053 bool
			}
			err := c.Post(`mutation { issue4053(input: {}) }`, &resp)
			assert.NoError(t, err)
			assert.True(t, resp.Issue4053)
		}, "should not panic when input2 is null")
	})

	t.Run("sending empty input2 should yield default values", func(t *testing.T) {
		resolver := &Stub{}
		resolver.MutationResolver.Issue4053 = func(ctx context.Context, input *Issue4053Input1) (bool, error) {
			require.NotNil(t, input, "input should not be nil")
			expected := Issue4053Input2{
				Hello:            "",
				HelloWithDefault: "world",
			}
			assert.Equal(t, expected, input.Input2,
				"input2 should have default values when empty input is passed")
			return true, nil
		}

		srv := handler.New(NewExecutableSchema(Config{Resolvers: resolver}))
		srv.AddTransport(transport.POST{})
		srv.SetRecoverFunc(nil) // disable panic recovery to allow test to fail if a panic occurs
		c := client.New(srv)

		require.NotPanics(t, func() {
			var resp struct {
				Issue4053 bool
			}
			err := c.Post(`mutation { issue4053(input: {input2: {}}) }`, &resp)
			assert.NoError(t, err)
			assert.True(t, resp.Issue4053)
		}, "should not panic when input2 is null")
	})
}
