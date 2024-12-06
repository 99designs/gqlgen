---
title: 'Enabling introspection'
description: Allow users to introspect schemas.
linkTitle: Introspection
menu: { main: { parent: 'reference', weight: 10 } }
---

One of the best features of GraphQL is it's powerful discoverability, known as [introspection][introspection]. Introspection allows clients to query the server's schema about itself, and is the foundation of many tools like GraphiQL and Apollo Studio.

## Enabling introspection

To enable introspection for the whole server, you use the bundled middleware extension `github.com/99designs/gqlgen/graphql/handler/extension.Introspection`:

```go
srv := handler.New(es)

// Add server setup.
srv.AddTransport(transport.Options{})
srv.AddTransport(transport.POST{})

// Add the introspection middleware.
srv.Use(extension.Introspection{})
```

To opt in to introspection for certain environments, you can just guard the middleware with an environment variable:

```go
srv := handler.New(es)

// Server setup...

if os.Getenv("ENVIRONMENT") == "development" {
    srv.Use(extension.Introspection{})
}
```

## Disabling introspection based on authentication

Introspection can also be guarded on a per-request context basis. For example, you can disable it in a middleware based on user authentication:

```go
srv := handler.New(es)

// Server setup...

srv.Use(extension.Introspection{})
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
    if !userForContext(ctx).IsAdmin {
        graphql.GetOperationContext(ctx).DisableIntrospection = true
    }

    return next(ctx)
})
```

[introspection]: https://graphql.org/learn/introspection/
