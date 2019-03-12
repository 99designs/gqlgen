---
title: 'Preventing overly complex queries'
description: Avoid denial of service attacks by calculating query costs and limiting complexity.
linkTitle: Query Complexity
menu: { main: { parent: 'reference' } }
---

GraphQL provides a powerful way to query your data, but putting great power in the hands of your API clients also exposes you to a risk of denial of service attacks. You can mitigate that risk with gqlgen by limiting the complexity of the queries you allow.

## Expensive Queries

Consider a schema that allows listing blog posts. Each blog post is also related to other posts.

```graphql
type Query {
	posts(count: Int = 10): [Post!]!
}

type Post {
	title: String!
	text: String!
	related(count: Int = 10): [Post!]!
}
```

It's not too hard to craft a query that will cause a very large response:

```graphql
{
	posts(count: 100) {
		related(count: 100) {
			related(count: 100) {
				related(count: 100) {
					title
				}
			}
		}
	}
}
```

The size of the response grows exponentially with each additional level of the `related` field. Fortunately, gqlgen's `http.Handler` includes a way to guard against this type of query.

## Limiting Query Complexity

Limiting query complexity is as simple as adding a parameter to the `handler.GraphQL` function call:

```go
func main() {
    c := Config{ Resolvers: &resolvers{} }
    gqlHandler := handler.GraphQL(
        blog.NewExecutableSchema(c),
        handler.ComplexityLimit(5), // This line is the key
    )
    http.Handle("/query", gqlHandler)
}
```

Now any query with complexity greater than 5 is rejected by the API. By default, each field and level of depth adds one to the overall query complexity. You can also use `handler.ComplexityLimitFunc` to dynamically configure the complexity limit per request.

This helps, but we still have a problem: the `posts` and `related` fields, which return arrays, are much more expensive to resolve than the scalar `title` and `text` fields. However, the default complexity calculation weights them equally. It would make more sense to apply a higher cost to the array fields.

## Custom Complexity Calculation

To apply higher costs to certain fields, we can use custom complexity functions.

```go
func main() {
    c := Config{ Resolvers: &resolvers{} }

    countComplexity := func(childComplexity, count int) int {
        return count * childComplexity
    }
    c.Complexity.Query.Posts = countComplexity
    c.Complexity.Post.Related = countComplexity

    gqlHandler := handler.GraphQL(
        blog.NewExecutableSchema(c),
        handler.ComplexityLimit(100),
    )
    http.Handle("/query", gqlHandler)
}
```

When we assign a function to the appropriate `Complexity` field, that function is used in the complexity calculation. Here, the `posts` and `related` fields are weighted according to the value of their `count` parameter. This means that the more posts a client requests, the higher the query complexity. And just like the size of the response would increase exponentially in our original query, the complexity would also increase exponentially, so any client trying to abuse the API would run into the limit very quickly.

By applying a query complexity limit and specifying custom complexity functions in the right places, you can easily prevent clients from using a disproportionate amount of resources and disrupting your service.
