---
linkTitle: Subscriptions
title: Subscription Context Field
description: How to propagate per-event context in GraphQL subscriptions using subscription_context_field.
menu: { main: { parent: 'reference', weight: 10 } }
---

## Overview

By default, subscription resolvers in gqlgen return a plain channel:

```go
func (r *subscriptionResolver) MessageAdded(ctx context.Context, roomName string) (<-chan *Message, error) {
    ch := make(chan *Message, 1)
    // ... populate ch ...
    return ch, nil
}
```

Each subscription event is a value on that channel. The response context (e.g. for tracing spans, per-event metadata) is the same `ctx` that was passed to the resolver.

This works well for most use cases. However, interceptors (middlewares registered via `AroundResponses`) only receive the original request context — they cannot access context information that is specific to an individual subscription event.

## Enabling `subscription_context_field`

When you set `subscription_context_field: true` in your `gqlgen.yml`, each subscription event can carry its own context. Resolvers return a channel of `graphql.SubscriptionField[T]` instead of `<-chan T`:

```yaml
# gqlgen.yml
subscription_context_field: true
```

The generated resolver interface changes from:

```go
// Default (disabled)
MessageAdded(ctx context.Context, roomName string) (<-chan *Message, error)
```

to:

```go
// Enabled
MessageAdded(ctx context.Context, roomName string) (<-chan graphql.SubscriptionField[*Message], error)
```

### Implementing the resolver

Use `graphql.NewSubscriptionField` to wrap each event with its own context:

```go
func (r *subscriptionResolver) MessageAdded(ctx context.Context, roomName string) (<-chan graphql.SubscriptionField[*Message], error) {
    ch := make(chan graphql.SubscriptionField[*Message], 1)

    go func() {
        defer close(ch)
        for {
            select {
            case msg := <-r.events:
                // Each event carries its own context (e.g. with a tracing span)
                eventCtx, span := tracer.Start(ctx, "subscription.event")
                ch <- graphql.NewSubscriptionField(eventCtx, msg)
                span.End()
            case <-ctx.Done():
                return
            }
        }
    }()

    return ch, nil
}
```

### Accessing the event context in interceptors

When `subscription_context_field` is enabled, each `graphql.Response` emitted by a subscription has its `Context` field populated with the per-event context you provided:

```go
srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
    resp := next(ctx)
    if resp != nil && resp.Context != nil {
        // resp.Context is the per-event context from graphql.NewSubscriptionField
        span := trace.SpanFromContext(resp.Context)
        // ... use span ...
    }
    return resp
})
```

For queries and mutations, `graphql.Response.Context` is always set to the request context.

## The `graphql.SubscriptionField[T]` interface

```go
type SubscriptionField[T any] interface {
    GetContext() context.Context
    GetField() T
}
```

Use `graphql.NewSubscriptionField(ctx, value)` to create an instance.

## Breaking changes

Enabling `subscription_context_field: true` is a **breaking change** for existing code:

1. All subscription resolvers must be updated to return `<-chan graphql.SubscriptionField[T]` instead of `<-chan T`.
2. Run `gqlgen generate` after adding the flag — the generated interfaces will be updated automatically.
3. Update all resolver implementations to use `graphql.NewSubscriptionField(ctx, value)` when sending events.

This is why the feature is **disabled by default** and must be explicitly opted into.

## Summary

| | Default (`false`) | Enabled (`true`) |
|---|---|---|
| Resolver return type | `(<-chan T, error)` | `(<-chan graphql.SubscriptionField[T], error)` |
| `graphql.Response.Context` | Request context | Per-event context |
| Per-event tracing | Not available | Available |
| Backward compatible | Yes | No (requires resolver updates) |
