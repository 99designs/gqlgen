---
title: 'Determining which fields were requested by a query'
description: How to determine which fields a query requested in a resolver.
linkTitle: Field Collection
menu: { main: { parent: 'reference', weight: 10 } }
---

Often it is useful to know which fields were queried for in a resolver.  Having this information can allow a resolver to only fetch the set of fields required from a data source, rather than over-fetching everything and allowing gqlgen to do the rest.

This process is known as [Field Collection](https://spec.graphql.org/draft/#sec-Field-Collection) — gqlgen automatically does this in order to know which fields should be a part of the response payload.  The set of collected fields does however depend on the type being resolved.  Queries can contain fragments, and resolvers can return interfaces and unions, therefore the set of collected fields cannot be fully determined until the type of the resolved object is known.

Within a resolver, there are several API methods available to query the selected fields.

## CollectAllFields

`CollectAllFields` is the simplest way to get the set of queried fields.  It will return a slice of strings of the field names from the query.  This will be a unique set of fields, and will return all fragment fields, ignoring fragment Type Conditions.

Given the following example query:

```graphql
query {
    foo {
        fieldA
        ... on Bar {
            fieldB
        }
        ... on Baz {
            fieldC
        }
    }
}
```

Calling `CollectAllFields` from a resolver will yield a string slice containing `fieldA`, `fieldB`, and `fieldC`.

## CollectFieldsCtx

`CollectFieldsCtx` is useful in cases where more information on matches is required, or the set of collected fields should match fragment type conditions for a resolved type.  `CollectFieldsCtx` takes a `satisfies` parameter, which should be a slice of strings of types that the resolved type will satisfy.

For example, given the following schema:

```graphql
interface Shape {
    area: Float
}
type Circle implements Shape {
    radius: Float
    area: Float
}
union Shapes = Circle
```

The type `Circle` would satisfy `Circle`, `Shape`, and `Shapes` — these values should be passed to `CollectFieldsCtx` to get the set of collected fields for a resolved `Circle` object.

> Note
>
> `CollectFieldsCtx` is just a convenience wrapper around `CollectFields` that calls the later with the selection set automatically passed through from the resolver context.

## FieldRequested

`FieldRequested` checks whether a specific field was requested in the current resolver's selection set. It respects `@skip`/`@include` directives and works with inline fragments, named fragment spreads, and aliases.

This is particularly useful when you want to conditionally fetch related data as part of the parent query (e.g. a SQL JOIN) rather than in a separate field-level resolver. A field-level resolver always requires a separate round-trip to the database. When the relationship doesn't cleanly map to an independent query — or when combining the data into a single query is more efficient — `FieldRequested` lets you conditionally include that data without fetching things the client never asked for.

```golang
func (r *queryResolver) User(ctx context.Context, id int) (*User, error) {
    includeReviews := graphql.FieldRequested(ctx, "reviews")
    return fetchUser(id, includeReviews)  // JOIN reviews only if requested
}
```

### Dot-notation for nested fields

`FieldRequested` supports dot-notation to check nested fields. The path is **relative to the current resolver's selection set** — not the root of the query.

For example, given this query:

```graphql
query {
    user(id: 1) {
        id
        reviews {
            body
            author {
                name
            }
        }
    }
}
```

Inside the `User` resolver, the selection set starts at the fields under `user`. So:

| Call | Result | Why |
|------|--------|-----|
| `graphql.FieldRequested(ctx, "id")` | `true` | `id` is in the selection set |
| `graphql.FieldRequested(ctx, "reviews")` | `true` | `reviews` is in the selection set |
| `graphql.FieldRequested(ctx, "reviews.body")` | `true` | `body` is under `reviews` |
| `graphql.FieldRequested(ctx, "reviews.author")` | `true` | `author` is under `reviews` |
| `graphql.FieldRequested(ctx, "reviews.author.name")` | `true` | `name` is under `reviews.author` |
| `graphql.FieldRequested(ctx, "reviews.author.email")` | `false` | `email` was not requested |
| `graphql.FieldRequested(ctx, "user")` | `false` | `user` is the parent — not in this selection set |

## AnyFieldRequested

`AnyFieldRequested` returns true if **any** of the given field paths were requested. This is useful when multiple fields represent the same unit of work — for example, `reviews`, `reviewCount`, and `averageRating` might all require the same SQL JOIN:

```golang
if graphql.AnyFieldRequested(ctx, "reviews", "reviewCount", "averageRating") {
    // JOIN reviews table — needed by any of these fields
}
```

Both functions work with inline fragments, named fragment spreads, aliases, and federation entity resolvers.

## Practical example

Say we have the following GraphQL query

```graphql
query {
  flowBlocks {
    id
    block {
      id
      title
      type
      choices {
        id
        title
        description
        slug
      }
    }
  }
}
```

We don't want to overfetch our database so we want to know which field are requested.
Here is an example which get's all requested field as convenient string slice, which can be easily checked.

```golang
func GetPreloads(ctx context.Context) []string {
	return GetNestedPreloads(
		graphql.GetOperationContext(ctx),
		graphql.CollectFieldsCtx(ctx, nil),
		"",
	)
}

func GetNestedPreloads(ctx *graphql.OperationContext, fields []graphql.CollectedField, prefix string) (preloads []string) {
	for _, column := range fields {
		prefixColumn := GetPreloadString(prefix, column.Name)
		preloads = append(preloads, prefixColumn)
		preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.Selections, nil), prefixColumn)...)
	}
	return
}

func GetPreloadString(prefix, name string) string {
	if len(prefix) > 0 {
		return prefix + "." + name
	}
	return name
}

```

So if we call these helpers in our resolver:
```golang
func (r *queryResolver) FlowBlocks(ctx context.Context) ([]*FlowBlock, error) {
	preloads := GetPreloads(ctx)
```
it will result in the following string slice:
```
["id", "block", "block.id", "block.title", "block.type", "block.choices", "block.choices.id", "block.choices.title", "block.choices.description", "block.choices.slug"]
```
