---
title: "Dynamically requiring fields with an executable directive"
description: Let clients opt individual fields into non-null (required) semantics at runtime using a @priority directive and graphql.MarkNonNull.
linkTitle: Dynamic required fields
menu: { main: { parent: 'recipes' } }
---

In GraphQL, [null propagation](https://spec.graphql.org/October2021/#sec-Handling-Field-Errors)
(nulling out a parent object when one of its children fails) only applies to
fields declared as non-null (`!`) in the schema. This is normally a
compile-time property: a field is either `String` or `String!`, and that never
changes.

gqlgen exposes `graphql.MarkNonNull(ctx)`, a runtime primitive that lets a
field middleware opt a *nullable* field into non-null semantics **for a single
request**. When a marked field resolves to `nil` (or returns an error), the
error propagates and the nearest nullable ancestor is set to `null`, exactly as
if the field had been declared with a trailing `!`.

The recommended way to expose this is through an **executable directive** that
the client writes into the query - for example `@priority(value: REQUIRED)`.
This keeps the behavior explicit and client-driven, in the same spirit as the
built-in `@skip` and `@include` directives.

> **Warning**
>
> Using `MarkNonNull` (or any executable directive) to alter null-propagation
> semantics makes the response shape depend on runtime logic rather than the
> schema. A field declared as nullable (`String`) can behave like a non-null
> field (`String!`) for a given request. This behavior:
>
> - **cannot be discovered via introspection**, and
> - **cannot be validated statically** by client tooling (GraphQL IDEs, type
>   generators, query validators).
>
> In other words, the schema is no longer the single source of truth for the
> client-server contract. Use this only when you control **both** the client
> and the server, and when the benefit (e.g. stricter data integrity for
> critical fields) outweighs the loss of schema-time guarantees. **Avoid it on
> public APIs or any API consumed by third-party clients.**

## When to use this

Good fits:

- Internal microservice-to-microservice calls where both ends are owned by the
  same team.
- Data-critical clients that would rather receive an explicit error (and a
  nulled-out parent) than a partial object with a missing critical field.

When **not** to use it:

- Public APIs or any schema consumed by third-party clients.
- Anywhere clients rely on introspection-based codegen or static validation to
  reason about which fields can be null.
- As a substitute for actually declaring a field non-null in the schema. If a
  field is *always* required, declare it `!` in the schema instead; that is
  introspectable and validated.

## Declare the directive in the schema

Define an executable directive (one whose location is `FIELD`, so it can appear
in queries) and an enum for its argument:

```graphql
enum Priority {
  OPTIONAL
  REQUIRED
}

"Marks a field as semantically required for this request."
directive @priority(value: Priority!) on FIELD
```

Apply it to fields that are nullable in the schema but that a particular client
wants to treat as required:

```graphql
type Query {
  user(id: ID!): User
}

type User {
  id: ID!
  name: String!
  # nullable in the schema; clients may opt into requiring them per-request
  nickname: String
  avatarURL: String
}
```

After editing the schema, run `go generate ./...`. gqlgen adds the directive to
the `DirectiveRoot` and generates the field middleware that invokes it:

```go
type DirectiveRoot struct {
	Priority func(ctx context.Context, obj any, next graphql.Resolver, value model.Priority) (res any, err error)
}
```

## Implement and register the directive

The directive handler runs as part of the field's resolver middleware chain. It
calls `graphql.MarkNonNull(ctx)` when the client asked for `REQUIRED`, then
calls `next(ctx)` to run the resolver as usual:

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/[username]/gqlgen-todos/graph"
	"github.com/[username]/gqlgen-todos/graph/model"
)

func main() {
	c := graph.Config{Resolvers: &graph.Resolver{}}

	c.Directives.Priority = func(ctx context.Context, obj any, next graphql.Resolver, value model.Priority) (any, error) {
		if value == model.PriorityRequired {
			// Opt this field into non-null semantics for THIS request only.
			// If the resolver below returns nil or an error, gqlgen emits a
			// "must not be null" error and propagates null up to the nearest
			// nullable ancestor - exactly as if the field were declared "!".
			graphql.MarkNonNull(ctx)
		}
		return next(ctx)
	}

	srv := handler.New(graph.NewExecutableSchema(c))
	srv.AddTransport(transport.POST{})

	http.Handle("/query", srv)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`MarkNonNull` only sets a flag on the current field's context; it must be called
before the resolver returns (calling it before `next(ctx)`, as above, is the
simplest correct place).

## Behavior with and without the directive

Assume `nickname` resolves to `nil` (the user has no nickname set, or its
resolver errors).

**Without the directive** - default schema behavior. `nickname` is nullable, so
it is simply `null` and the rest of the object is returned:

```graphql
{
  user(id: "1") {
    name
    nickname
  }
}
```

```json
{
  "data": {
    "user": { "name": "Alice", "nickname": null }
  }
}
```

**With the directive** - the client opts `nickname` into required semantics.
Because the field is now treated as non-null and resolved to `nil`, the error
propagates and the nearest nullable ancestor (`user`) becomes `null`:

```graphql
{
  user(id: "1") {
    name
    nickname @priority(value: REQUIRED)
  }
}
```

```json
{
  "data": {
    "user": null
  },
  "errors": [
    {
      "message": "must not be null",
      "path": ["user", "nickname"]
    }
  ]
}
```

The only difference between the two requests is the directive the client chose
to include. The server's schema and resolvers are identical.

## How it works

When the schema declares any `on FIELD` directive, gqlgen wraps every field
resolver with a generated `_fieldMiddleware`. For each field in the incoming
query it looks at the directives the client attached to that field and calls the
matching handler. Your `@priority` handler calls `graphql.MarkNonNull(ctx)`,
which sets `FieldContext.NonNull = true`.

When the resolver returns, gqlgen checks that flag. For a marked-but-nullable
field that resolved to `nil`, it emits the standard `must not be null` error and
returns the internal `graphql.RequiredNull` sentinel. The generated parent code
recognises that sentinel and triggers the same null-propagation cascade that a
schema-level `!` violation would. Statically non-null fields are unaffected and
keep their existing behavior.

## Alternative: a global field interceptor

If you would rather not declare a directive, you can call `MarkNonNull` from a
global field interceptor (an `AroundFields` middleware) based on whatever logic
you like. **This is the "hidden server logic" path the warning above cautions
against** - the client has no way to see or opt out of it - so prefer the
directive approach unless you have a specific reason not to.

```go
srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
	fc := graphql.GetFieldContext(ctx)
	if shouldRequire(fc) { // your own logic
		graphql.MarkNonNull(ctx)
	}
	return next(ctx)
})
```

## Future work: introspection

There is currently no way to advertise these dynamic constraints through
introspection; by design, the requirement only exists for the duration of a
request that opts into it. A future enhancement could expose "dynamic
constraints" through a custom introspection extension, but that is out of scope
for this feature today.
