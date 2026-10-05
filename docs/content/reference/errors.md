---
linkTitle: Handling Errors
title: Sending custom error data in the graphql response
description: Customising graphql error types to send custom error data back to the client using gqlgen.
menu: { main: { parent: 'reference', weight: 10 } }
---

## Returning errors

All resolvers simply return an error to be sent to the user. The assumption is that any error message returned
here is appropriate for end users. If certain messages aren't safe, customise the error presenter.

### Multiple errors

To return multiple errors you can call the `graphql.Error` functions like so:

```go
package foo

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/99designs/gqlgen/graphql"
)

// DoThings add errors to the stack.
func (r Query) DoThings(ctx context.Context) (bool, error) {
	// Print a formatted string
	graphql.AddErrorf(ctx, "Error %d", 1)

	// Pass an existing error out
	graphql.AddError(ctx, gqlerror.Errorf("zzzzzt"))

	// Or fully customize the error
	graphql.AddError(ctx, &gqlerror.Error{
		Path:       graphql.GetPath(ctx),
		Message:    "A descriptive error message",
		Extensions: map[string]interface{}{
			"code": "10-4",
		},
	})

	// And you can still return an error if you need
	return false, gqlerror.Errorf("BOOM! Headshot")
}
```

They will be returned in the same order in the response, eg:
```json
{
  "data": {
    "todo": null
  },
  "errors": [
    { "message": "Error 1", "path": [ "todo" ] },
    { "message": "zzzzzt", "path": [ "todo" ] },
    { "message": "A descriptive error message", "path": [ "todo" ], "extensions": { "code": "10-4" } },
    { "message": "BOOM! Headshot", "path": [ "todo" ] }
  ]
}
```

or you can simply return multiple errors

```go
package foo

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/99designs/gqlgen/graphql"
)

var errSomethingWrong = errors.New("some validation failed")

// DoThingsReturnMultipleErrors collect errors and returns it if any.
func (r Query) DoThingsReturnMultipleErrors(ctx context.Context) (bool, error) {
	errList := gqlerror.List{}

	// Add existing error
	errList = append(errList, gqlerror.Wrap(errSomethingWrong))

	// Create new formatted and append
	errList = append(errList, gqlerror.Errorf("invalid value: %s", "invalid"))

	// Or fully customize the error and append
	errList = append(errList, &gqlerror.Error{
		Path:       graphql.GetPath(ctx),
		Message:    "A descriptive error message",
		Extensions: map[string]interface{}{
			"code": "10-4",
		},
	})

	return false, errList
}
```

They will be returned in the same order in the response, eg:
```json
{
  "data": {
    "todo": null
  },
  "errors": [
    { "message": "some validation failed", "path": [ "todo" ] },
    { "message": "invalid value: invalid", "path": [ "todo" ] },
    { "message": "A descriptive error message", "path": [ "todo" ], "extensions": { "code": "10-4" } },
  ]
}
```

### Returning errors from extensions

When writing a HandlerExtension (such as for rate limiting or authentication),
you might want to short-circuit the request and return an error without executing
the query.

When doing so, you must wrap your error response using `graphql.OneShot`.
Since streaming transports like WebSockets or Server-Sent Events expect a stream
of responses ending in `nil`, returning a bare closure that always yields the
error will cause the transport to loop infinitely and spam the client, or
cause high CPU consumption and flood logs.

```go
package foo

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type MyInterceptor struct{}

func (m MyInterceptor) ExtensionName() string {
	return "MyInterceptor"
}

func (m MyInterceptor) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (m MyInterceptor) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	if !isAuthorized(ctx) {
		err := gqlerror.Errorf("unauthorized")
		// ALWAYS use graphql.OneShot when short-circuiting a request!
		return graphql.OneShot(graphql.ErrorResponse(ctx, err.Message))
	}
	return next(ctx)
}

func isAuthorized(ctx context.Context) bool {
    return false // implement authorization logic here
}
```

## Hooks

### The error presenter

All `errors` returned by resolvers, or from validation, pass through a hook before being displayed to the user.
This hook gives you the ability to customise errors however makes sense in your app.

The default error presenter will capture the resolver path and use the Error() message in the response.

You change this when creating the server:
```go
package bar

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func main() {
	server := handler.New(MakeExecutableSchema(resolvers))

	// Server setup...

	server.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		err := graphql.DefaultErrorPresenter(ctx, e)

		var myErr *MyError
		if errors.As(e, &myErr) {
			err.Message = "Eeek!"
		}

		return err
	})
}

```

This function will be called with the same resolver context that generated it, so you can extract the
current resolver path and whatever other state you might want to notify the client about.


### Non-null violations

When a field the schema declares non-null (`String!`) resolves to null, gqlgen raises a field
error and propagates the null to the nearest nullable ancestor, as the GraphQL specification
requires. The error it raises is a `*graphql.InvalidNullError`, which carries the diagnosis as
fields rather than only as message text:

```go
server.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
	err := graphql.DefaultErrorPresenter(ctx, e)

	if nullErr, ok := errors.AsType[*graphql.InvalidNullError](e); ok {
		err.Extensions = map[string]any{
			"code":  "INVALID_NULL",
			"field": nullErr.SchemaCoordinate(), // "OuterObject.inner"
			"type":  nullErr.DeclaredType,       // "InnerObject!"
			"from":  string(nullErr.Source),     // "model_field"
		}
	}

	return err
})
```

`errors.Is(e, graphql.ErrInvalidNull)` matches any such violation when you only need to know that one
happened.

`SchemaCoordinate()` returns the field's
[schema coordinate](https://spec.graphql.org/September2025/#sec-Schema-Coordinates). It names the
**concrete** object type, which the error path cannot: for a field reached through an interface the
path shows `["node","child"]`, while the coordinate shows `ConcreteNodeA.child`. `Source` says
where to look in your Go code:

| `Source`           | Meaning                                                                      |
| ------------------ | ---------------------------------------------------------------------------- |
| `resolver`         | A resolver method returned nil.                                              |
| `model_method`     | A method on the model returned nil.                                          |
| `model_field`      | A model struct field was nil, and the field has no resolver.                 |
| `marshaler`        | A marshaler mapped a non-nil value to null (`MarshalTime` on the zero time). |
| `runtime_non_null` | `graphql.MarkNonNull` raised a nullable field that then resolved to nil.     |

`ListElement` reports that the null was an element of a list rather than the field's own value;
`Path` is the response path, including list indices.

The text of `Error()` is intended for developers and is not part of the API — match on the type or
the sentinel rather than on the message.

#### Structured logging

`InvalidNullError` also answers `Attrs() []slog.Attr`, so logging middleware can record the diagnosis
without importing gqlgen's type. Walk the chain and ask each error for its attributes:

```go
func attrs(err error) []slog.Attr {
	var out []slog.Attr
	for e := err; e != nil; e = errors.Unwrap(e) {
		if a, ok := e.(interface{ Attrs() []slog.Attr }); ok {
			out = append(out, a.Attrs()...)
		}
	}
	return out
}
```

A violation yields `gql.field`, `gql.type`, `gql.null_source`, `gql.path`, and `gql.list_element`
when it applies. Empty values are omitted. Because `gqlerror.List` unwraps to its members, a
collector that follows multi-error branches gathers the attributes for every violation in one
response.

### The panic handler

There is also a panic handler, called whenever a panic happens to gracefully return a message to the user before
stopping parsing. This is a good spot to notify your bug tracker and send a custom message to the user. Any errors
returned from here will also go through the error presenter.

You change this when creating the server:
```go
server := handler.New(MakeExecutableSchema(resolvers))

// Server setup...

server.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
    // Notify bug tracker...

		return gqlerror.Errorf("Internal server error!")
})
```

While these handlers are useful in production to make sure the program does not crash, even if a user finds an issue that causes a crash-condition. During development, it can sometimes be more useful to properly crash, potentially generating a coredump to [enable further debugging](https://go.dev/wiki/CoreDumpDebugging).

To allow your program to crash on a panic, add this to your config file:

```yaml
omit_panic_handler: true
```
