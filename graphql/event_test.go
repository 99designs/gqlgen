package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamWithoutEventContext(t *testing.T) {
	inputCtx := context.WithValue(context.Background(), eventCtxKey("k"), "v")

	adapted := StreamWithoutEventContext(func(context.Context) Marshaler {
		return MarshalString("payload")
	})

	gotCtx, m := adapted(inputCtx)
	assert.Equal(t, inputCtx, gotCtx, "the input context is threaded through unchanged")
	assertMarshalsTo(t, m, `"payload"`)
}

func TestSubscriptionEventResponseHandler(t *testing.T) {
	eventCtx := context.WithValue(context.Background(), eventCtxKey("k"), "v")
	next := func(context.Context) (context.Context, Marshaler) {
		return eventCtx, MarshalString("hi")
	}

	gotCtx, resp := SubscriptionEventResponseHandler(next)(context.Background())
	require.NotNil(t, resp)
	assert.Equal(t, eventCtx, gotCtx, "the per-event context is carried to the caller")
	assert.JSONEq(t, `"hi"`, string(resp.Data))
}

func TestSubscriptionEventResponseHandler_EndOfStream(t *testing.T) {
	inputCtx := context.WithValue(context.Background(), eventCtxKey("k"), "v")
	next := func(ctx context.Context) (context.Context, Marshaler) {
		return ctx, nil
	}

	gotCtx, resp := SubscriptionEventResponseHandler(next)(inputCtx)
	assert.Nil(t, resp, "a nil marshaler signals end-of-stream as a nil response")
	assert.Equal(t, inputCtx, gotCtx)
}

func TestSubscriptionResponseHandler(t *testing.T) {
	next := func(context.Context) (context.Context, Marshaler) {
		return context.Background(), MarshalString("hi")
	}

	resp := SubscriptionResponseHandler(next)(context.Background())
	require.NotNil(t, resp)
	assert.JSONEq(t, `"hi"`, string(resp.Data))

	endOfStream := func(ctx context.Context) (context.Context, Marshaler) {
		return ctx, nil
	}
	assert.Nil(t, SubscriptionResponseHandler(endOfStream)(context.Background()))
}
