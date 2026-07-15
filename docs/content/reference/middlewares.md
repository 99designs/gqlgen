---
linkTitle: Middlewares
title: Middlewares and Interceptors
description: How to hook into the GraphQL execution lifecycle using middlewares and interceptors.
menu: { main: { parent: 'reference', weight: 10 } }
---

## Overview

gqlgen provides a robust set of hooks to intercept and modify the GraphQL execution lifecycle. These are commonly referred to as **middlewares** or **interceptors**. They are particularly useful for cross-cutting concerns like:

- Authentication and Authorization
- Logging and Tracing
- Query complexity and rate limiting
- Error reporting

You can register middlewares directly on your `handler.Server` instance.

## Field Middleware (`AroundFields`)

Field middleware runs for *every* field in a GraphQL query that is resolved. It is highly granular and is the perfect place to enforce field-level permissions or log resolver execution times.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func main() {
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		rc := graphql.GetFieldContext(ctx)

		// Example: Logging field execution time
		start := time.Now()
		res, err = next(ctx)

		fmt.Printf("Field %s.%s took %v\n", rc.Object, rc.Field.Name, time.Since(start))

		return res, err
	})

	// ... continue server setup
}
```

## Operation Middleware (`AroundOperations`)

Operation middleware runs once for the entire GraphQL operation (Query,
Mutation, or Subscription). It is commonly used for request-level validation,
authenticating the user before any resolvers run, or logging the entire query
payload.

```go
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	oc := graphql.GetOperationContext(ctx)
	fmt.Printf("Incoming operation: %s\n", oc.OperationName)

	// Example: Reject unauthenticated requests
	if !isAuthorized(ctx) {
		// Use graphql.OneShot to ensure the error is sent exactly once!
		return graphql.OneShot(graphql.ErrorResponse(ctx, "unauthorized"))
	}

	// Continue executing the operation
	return next(ctx)
})
```

### Short-circuiting Requests (Important!)

If you want to reject a request inside an operation middleware (e.g., the user
is not authenticated), you might be tempted to simply return an error. However,
GraphQL handles streaming responses (like Subscriptions over WebSockets) where
the transport iterates over responses until it receives `nil`.

If you return an error without executing `next()`, you **must** wrap it in
`graphql.OneShot`. Failing to do so will cause streaming transports to loop
infinitely and spam the client with the same error!

### Example: Setting the Worker Limit Per Request

The `worker_limit` in `gqlgen.yml` is only a codegen-time default for how many
goroutines gqlgen uses when marshaling slices concurrently. You can override it
per request from an operation middleware by calling `SetWorkerLimit` on the
operation context (`0` means unlimited):

```go
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	oc := graphql.GetOperationContext(ctx)

	// Throttle lower-priority requests to a small pool of workers; everything
	// else falls back to the server-wide default / codegen worker_limit.
	if oc.Headers.Get("X-Priority") == "low" {
		oc.SetWorkerLimit(2)
	}

	return next(ctx)
})
```

`SetWorkerLimit` assigns a fresh value on this request's operation context only,
so it never affects other in-flight requests. Precedence is
**per-request override > server-wide `srv.SetWorkerLimit` > codegen `worker_limit`**.

## Root Field Middleware (`AroundRootFields`)

Root field middleware is similar to field middleware, but it only runs for the
root fields defined on your `Query`, `Mutation`, or `Subscription` types. This
is useful if you want to apply logic specifically at the entry points of your
graph without incurring the performance overhead of running it on every single
nested field.

```go
srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
	rc := graphql.GetRootFieldContext(ctx)
	fmt.Printf("Executing root field: %s\n", rc.Field.Name)

	return next(ctx)
})
```

## Response Middleware (`AroundResponses`)

Response middleware hooks into the very end of the execution lifecycle, just
before the response is serialized and sent back to the client. This can be run
multiple times per operation if the operation is a Subscription.

```go
srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	// Execute the operation and get the response
	resp := next(ctx)

	if resp != nil && len(resp.Errors) > 0 {
		fmt.Printf("Operation finished with %d errors\n", len(resp.Errors))
	}

	return resp
})
```

## Advanced: Handler Extensions

For more complex plugins, gqlgen provides the `graphql.HandlerExtension`
interface. Extensions can hook into multiple parts of the lifecycle at once
(parameter mutation, context mutation, operations, and fields).

Built-in features like Automatic Persisted Queries (APQ) and Apollo Tracing
are implemented as handler extensions. You can register an extension using
`srv.Use()`:

```go
srv.Use(extension.FixedComplexityLimit(50))
```

To create your own, implement `graphql.HandlerExtension` and the
specific interceptor interfaces (e.g., `graphql.OperationInterceptor`,
`graphql.FieldInterceptor`) you need.

### Example: A Worker Limit Extension

Extensions that implement `graphql.OperationContextMutator` receive the
`*graphql.OperationContext` directly, which makes them a clean place to set the
concurrent-slice worker limit per request. This overrides both the codegen
`worker_limit` and any server-wide `srv.SetWorkerLimit` default:

```go
package main

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// WorkerLimitByOperation caps concurrency for specific, known-expensive
// operations while leaving everything else on the default.
type WorkerLimitByOperation struct{}

// Assert we implement the interfaces we rely on.
var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = WorkerLimitByOperation{}

func (WorkerLimitByOperation) ExtensionName() string { return "WorkerLimitByOperation" }

func (WorkerLimitByOperation) Validate(graphql.ExecutableSchema) error { return nil }

func (WorkerLimitByOperation) MutateOperationContext(ctx context.Context, oc *graphql.OperationContext) *gqlerror.Error {
	if oc.OperationName == "ExpensiveReport" {
		oc.SetWorkerLimit(4) // 0 would mean unlimited
	}
	return nil
}
```

Register it on the server with `srv.Use`:

```go
srv.Use(WorkerLimitByOperation{})
```
