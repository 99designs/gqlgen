---
title: 'Disabling introspection'
description: Prevent users from introspecting schemas in production.
linkTitle: Introspection
menu: { main: { parent: 'reference' } }
---

One of the most powerful features of running graphql is its amazing discoverability, but sometimes you might not want to allow others to discover your endpoints.

## Disable it for the whole server

The easiest way to turn it on and off at runtime by passing a handler option when starting the server:

```go
srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.IntrospectionEnabled(false),
			),
		)
```

## Disabling based on authentication

Introspection can be enabled on a per-request context basis, so you can modify it in middleware based on user authentication too:
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
