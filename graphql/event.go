package graphql

import "context"

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
