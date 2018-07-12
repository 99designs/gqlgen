---
linkTitle: Handling Errors
title: Sending custom error data in the graphql response
description: Customising graphql error types to send custom error data back to the client using gqlgen.
menu: main
---

## Returning errors

All resolvers simply return an error to send it to the user. Its assumed that any error message returned
here is safe for users, if certain messages arent safe, customize the error presenter.

### Multiple errors

To return multiple errors you can call the `graphql.Error` functions like so:

```go
func (r Query) DoThings(ctx context.Context) (bool, error) {
	// Print a formatted string
	graphql.AddErrorf(ctx, "Error %d", 1)

	// Pass an existing error out
	graphql.AddError(ctx, err)

	// Fully customize the error, bypassing the presenter. You
	// wont get path information unless you add it yourself
	graphql.AddGraphqlError(ctx, &graphql.Error{
		Message: "A descriptive error message",
		Path: GetResolverContext(ctx).Path,
		Extensions: map[string]interface{}{
			"code": "10-4",
		}
	})

	// And you can still return an error if you need
	return false, errors.New("BOOM! Headshot")
}
```

## Hooks

### The error presenter

All `errors` returned by resolvers, or from validation pass through a hook before being displayed to the user.
This hook gives you the ability to customize errors however makes sense in your app.

The default error presenter will capture the resolver path and use the Error() message in the response. It will
also call an Extensions() method if one is present to return graphql extensions.

You change this when creating the handler:
```go
server := handler.GraphQL(MakeExecutableSchema(resolvers),
	handler.ErrorPresenter(
		func(ctx context.Context, e error) *graphql.Error {
			// any special logic you want to do here. This only
			// requirement is that it can be json encoded
			if myError, ok := e.(MyError) ; ok {
				return &graphql.Error{Message: "Eeek!"}
			}

			return graphql.DefaultErrorPresenter(ctx, e)
		}
	),
)
```

This function will be called with the the same resolver context that threw generated it, so you can extract the
current resolver path and whatever other state you might want to notify the client about.


### The panic handler

There is also a panic handler, called whenever a panic happens to gracefully return a message to the user before
stopping parsing. This is a good spot to notify your bug tracker and send a custom message to the user. Any errors
returned from here will also go through the error presenter.

You change this when creating the handler:
```go
server := handler.GraphQL(MakeExecutableSchema(resolvers),
	handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
		// notify bug tracker...

		return fmt.Errorf("Internal server error!")
	}
}
```

