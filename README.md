# graphql-go

[![GoDoc](https://godoc.org/github.com/neelance/graphql-go?status.svg)](https://godoc.org/github.com/neelance/graphql-go)

## Goals:

* full support of GraphQL spec
* minimal API
* support for context.Context and OpenTracing
* early error detection at application startup by type-checking if the given resolver matches the schema 
* resolvers are purely based on method sets (e.g. it's up to you if you want to resolve a GraphQL interface with a Go interface or a Go struct)
* nice error messages (no internal panics, even with an invalid schema or resolver; please file a bug if you see an internal panic)
* panic handling (a panic in a resolver should not take down the whole app)
* parallel execution of resolvers
