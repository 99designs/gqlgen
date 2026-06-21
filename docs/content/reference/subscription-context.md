---
linkTitle: Subscription event context
title: Per-event context for subscriptions
description: How to propagate per-event context from subscription resolvers to AroundResponses interceptors using the @subscriptionContext directive.
menu: { main: { parent: 'reference', weight: 10 } }
---

## Overview

By default, a subscription resolver returns a plain channel:

```go
func (r *subscriptionResolver) MessageAdded(ctx context.Context, room string) (<-chan *Message, error) {
    ch := make(chan *Message, 1)
    // populate ch from somewhere
    return ch, nil
}
```

`AroundResponses` interceptors observe the **subscription's** request context for every payload — that is, the context that existed when the subscription started. There is no per-event surface.

`@subscriptionContext` is an opt-in schema directive that changes this for one field at a time. When a subscription field is annotated, the resolver returns `<-chan graphql.Event[T]` instead of `<-chan T`, and each `Event` carries its own context. The graphql executor threads that context into the `ctx` parameter that `AroundResponses` interceptors already receive — no new field on `graphql.Response`, no new interceptor signature.

## Opting in

Add the directive to your schema and to the schema's directive declarations:

```graphql
directive @subscriptionContext on FIELD_DEFINITION

type Subscription {
  messageAdded(room: String!): Message! @subscriptionContext
  presenceChanged: Presence!
}
```

`messageAdded` gets the per-event treatment; `presenceChanged` keeps the existing shape. The opt-in is per field — other subscriptions in the same project remain unchanged.

Run `gqlgen generate` and the resolver interface for the marked field becomes:

```go
type SubscriptionResolver interface {
    MessageAdded(ctx context.Context, room string) (<-chan graphql.Event[*Message], error)
    PresenceChanged(ctx context.Context) (<-chan *Presence, error)
}
```

## Publishing events with context

`graphql.Event[T]` is a plain struct with exported fields:

```go
type Event[T any] struct {
    Context context.Context
    Value   T
}
```

Build one per published event:

```go
func (r *subscriptionResolver) MessageAdded(
    ctx context.Context, room string,
) (<-chan graphql.Event[*Message], error) {
    ch := make(chan graphql.Event[*Message], 1)
    go func() {
        defer close(ch)
        for msg := range r.events {
            eventCtx, span := tracer.Start(ctx, "subscription.event")
            ch <- graphql.Event[*Message]{Context: eventCtx, Value: msg}
            span.End()
        }
    }()
    return ch, nil
}
```

### Contract

- `Event.Context` **must** be derived from the subscription request context (typically via `context.WithValue` or `context.WithCancel`). Replacing it with an unrelated `context.Background()` loses request-scoped values such as the authenticated user and trace IDs. The runtime does not enforce this; it is your contract with the engine.
- If `Event.Context` is nil, the engine falls back to the subscription request context for that event. Set it explicitly to avoid surprises.

## Reading the context from an interceptor

The interceptor signature is unchanged:

```go
srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
    // ctx is the per-event context that the resolver attached to this event.
    // For unmarked subscriptions and for queries/mutations, ctx is the
    // operation's request context.
    span := trace.SpanFromContext(ctx)
    // ... use span ...
    return next(ctx)
})
```

There is no `Response.Context` field to read. The per-event context flows through `ctx` — the parameter the interceptor already has.

## Resolution-time context contract

When a subscription field is marked, the resolver runs **before** the `AroundResponses` chain for each event. As a result:

- Interceptors can still enrich `ctx` via `next(ctx2)`. Subsequent links in the AroundResponses chain see `ctx2`.
- Interceptor enrichment **does not** influence the field-resolver work for the current event (the work has already happened by the time the interceptor sees the response). Enrichment intended for resolvers belongs on `AroundFields` or `AroundRootFields`.
- For unmarked subscriptions and for queries/mutations, the existing order is preserved: middleware wraps resolver work, so `next(ctx2)` does affect resolver context.

## Default vs marked: comparison

|                                        | Default `<-chan T`             | Marked `<-chan graphql.Event[T]`            |
| -------------------------------------- | ------------------------------ | ------------------------------------------- |
| Resolver return type                   | `<-chan T`                     | `<-chan graphql.Event[T]`                   |
| AroundResponses ctx                    | Subscription request ctx       | Per-event ctx attached by the resolver      |
| `graphql.Response.Context`             | absent                         | absent                                      |
| Schema opt-in                          | none                           | `@subscriptionContext` on the field         |
| Project-wide config                    | none                           | none                                        |
| Generated code for unmarked fields     | byte-identical                 | byte-identical                              |
| Per-event tracing / metadata           | not available                  | available                                   |

## Known limitations

- `@subscriptionContext` does not currently compose with the `SUBSCRIPTION`-location directive middleware. A field cannot be both `@subscriptionContext`-marked AND have a custom `@directive on SUBSCRIPTION` in the same operation; the runtime will use the marked-field path and skip the SUBSCRIPTION middleware.
- The contract that `Event.Context` must derive from the subscription request context is documentation, not enforcement. The runtime cannot inspect the parent chain of an arbitrary `context.Context`.
