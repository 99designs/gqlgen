---
title: 'Disabling introspection'
description: Prevent users from introspecting schemas in production.
linkTitle: Introspection
menu: { main: { parent: 'reference' } }
---

One of the best features of GraphQL is it's powerful discoverability, but sometimes you don't want to allow others to explore your endpoint.

## Disable introspection for the whole server

To turn introspection on and off at runtime, pass the `IntrospectionEnabled` handler option when starting the server:

```go
srv := httptest.NewServer(
	handler.GraphQL(
		NewExecutableSchema(Config{Resolvers: resolvers}),
		handler.IntrospectionEnabled(false),
	),
)
```

## Disabling introspection based on authentication

Introspection can also be enabled on a per-request context basis.  For example, you could modify it in a middleware based on user authentication:

```go
srv := httptest.NewServer(
	handler.GraphQL(
		NewExecutableSchema(Config{Resolvers: resolvers}),
		handler.RequestMiddleware(func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
			if userForContext(ctx).IsAdmin {
				graphql.GetRequestContext(ctx).DisableIntrospection = true
			}

			return next(ctx)
		}),
	),
)
```
