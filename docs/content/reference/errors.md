---
linkTitle: Handling Errors
title: Sending custom error data in the graphql response
description: Customising graphql error types to send custom error data back to the client using gqlgen.
menu: main
---

All errors raised by gqlgen pass through a hook before being displayed to the user. This hook gives you the ability to
customize errors however makes sense in your app.


You can set it when creating the handler:
```go
server := handler.GraphQL(MakeExecutableSchema(resolvers),
	handler.ErrorPresenter(
		func(ctx context.Context, e error) graphql.MarshalableError {
			// any special logic you want to do here. This only
			// requirement is that it can be json encoded
			if myError, ok := e.(MyError) ; ok {
				return e
			}

			return graphql.DefaultErrorPresenter(ctx, e)
		}
	),
)
```

This function is called in a defer, so the stack is still at the right location and you have access to context to get
the current resolver path. By customizing the result you can add custom properties to errors, or implement your own
error type that is passed directly through to the client.


