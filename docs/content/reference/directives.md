---
title: Using schema directives to implement permission checks
description: Implementing graphql schema directives in golang for permission checks.
linkTitle: Schema Directives
menu: { main: { parent: 'reference', weight: 10 } }
---

Directives are a bit like annotations in any other language. They give you a way to specify some behaviour without directly binding to the implementation. This can be really useful for cross cutting concerns like permission checks.

**Note**: The current directives implementation is still fairly limited, and is designed to cover the most common "field middleware" case.

## Declare it in the schema

Directives are declared in your schema, along with all your other types. Lets define a @hasRole directive:

```graphql
directive @hasRole(role: Role!) on FIELD_DEFINITION

enum Role {
    ADMIN
    USER
}
```

When we next run go generate, gqlgen will add this directive to the DirectiveRoot
```go
type DirectiveRoot struct {
	HasRole func(ctx context.Context, obj interface{}, next graphql.Resolver, role Role) (res interface{}, err error)
}
```

The arguments are:
 - *ctx*: the parent context
 - *obj*: the object containing the value this was applied to, eg:
    - for field definition directives, the object/input object that contains the field
    - for argument directives, a map containing all arguments
 - *next*: the next directive in the directive chain, or the field resolver. This should be called to get the
           value of the field/argument/whatever. You can block access to the field by not calling next for permission
           checks etc.
 - *...args*: Any args to the directive will be passed in too.

## Use it in the schema

We can call this on any field definition now:
```graphql
type Mutation {
	deleteUser(userID: ID!): Bool @hasRole(role: ADMIN)
}
```

## Implement the directive

Finally, we need to implement the directive, and pass it in when starting the server:
```go
package main

func main() {
	c := Config{ Resolvers: &resolvers{} }
	c.Directives.HasRole = func(ctx context.Context, obj interface{}, next graphql.Resolver, role Role) (interface{}, error) {
		if !getCurrentUser(ctx).HasRole(role) {
			// block calling the next resolver
			return nil, fmt.Errorf("Access denied")
		}

		// or let it pass through
		return next(ctx)
	}

	http.Handle("/query", handler.GraphQL(todo.NewExecutableSchema(c), ))
	log.Fatal(http.ListenAndServe(":8081", nil))
}
```
