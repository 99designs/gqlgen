package graphql

import (
	"bytes"
	"context"
)

// Event bundles a per-event context with a subscription payload value.
//
// Subscription resolvers may return <-chan Event[T] (instead of <-chan T) for
// fields marked with the @subscriptionContext directive. The Context carried
// by each Event is observed by AroundResponses interceptors as the ctx
// parameter of their handler when that event's response is dispatched.
//
// Context must be derived from the subscription request context (typically
// via context.WithValue or context.WithCancel). Replacing it with an
// unrelated background context loses request-scoped values such as the
// authenticated user and trace IDs. This is a contract, not enforced by the
// runtime.
//
// When Event.Context is nil, the engine falls back to the input context
// driving the subscription iteration; resolvers should set Context
// explicitly for every event they publish.
type Event[T any] struct {
	Context context.Context
	Value   T
}

// StreamWithoutEventContext adapts a plain subscription stream handler to the
// per-event-context shape. It is used when a Subscription type mixes
// @subscriptionContext fields with unmarked ones: the unmarked field publishes
// no per-event context, so every event is paired with the input context
// unchanged, letting both kinds of field share one dispatch path.
func StreamWithoutEventContext(
	next func(context.Context) Marshaler,
) func(context.Context) (context.Context, Marshaler) {
	return func(ctx context.Context) (context.Context, Marshaler) {
		return ctx, next(ctx)
	}
}

// SubscriptionEventResponseHandler builds a [ResponseHandlerWithContext] that
// marshals each event yielded by the subscription stream handler next into a
// *Response, carrying that event's context through to the caller so
// AroundResponses interceptors observe it. It yields a nil response once next
// signals end-of-stream with a nil marshaler.
func SubscriptionEventResponseHandler(
	next func(context.Context) (context.Context, Marshaler),
) ResponseHandlerWithContext {
	var buf bytes.Buffer
	return func(ctx context.Context) (context.Context, *Response) {
		buf.Reset()
		eventCtx, data := next(ctx)
		if data == nil {
			return ctx, nil
		}
		data.MarshalGQL(&buf)
		return eventCtx, &Response{Data: buf.Bytes()}
	}
}

// SubscriptionResponseHandler builds a plain [ResponseHandler] from the same
// subscription stream handler, discarding each event's context. It backs the
// default Exec path, where per-event context is not surfaced;
// [SubscriptionEventResponseHandler] backs ExecWithEventContext, which preserves it.
func SubscriptionResponseHandler(
	next func(context.Context) (context.Context, Marshaler),
) ResponseHandler {
	withContext := SubscriptionEventResponseHandler(next)
	return func(ctx context.Context) *Response {
		_, resp := withContext(ctx)
		return resp
	}
}

// NullStream returns a subscription field handler that yields [Null]. Generated
// subscription middleware uses it as the error/fallback result so the equivalent
// null closure is not duplicated into every generated executor.
func NullStream() func(context.Context) Marshaler {
	return func(context.Context) Marshaler {
		return Null
	}
}

// NullEventStream is the per-event-context counterpart of [NullStream], used when
// the subscription_context_field option is enabled: it yields the input context
// alongside [Null].
func NullEventStream() func(context.Context) (context.Context, Marshaler) {
	return func(ctx context.Context) (context.Context, Marshaler) {
		return ctx, Null
	}
}
