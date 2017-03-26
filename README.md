# graphql-go

[![Sourcegraph](https://sourcegraph.com/github.com/neelance/graphql-go/-/badge.svg)](https://sourcegraph.com/github.com/neelance/graphql-go?badge)
[![Build Status](https://semaphoreci.com/api/v1/neelance/graphql-go/branches/master/badge.svg)](https://semaphoreci.com/neelance/graphql-go)
[![GoDoc](https://godoc.org/github.com/neelance/graphql-go?status.svg)](https://godoc.org/github.com/neelance/graphql-go)

## Status

The project is under heavy development. It is stable enough so we use it in production at [Sourcegraph](https://sourcegraph.com), but expect changes.

## Goals

* [ ] full support of [GraphQL spec (October 2016)](https://facebook.github.io/graphql/)
  * [ ] propagation of `null` on resolver errors
  * [x] everything else
* [x] minimal API
* [x] support for context.Context and OpenTracing
* [x] early error detection at application startup by type-checking if the given resolver matches the schema 
* [x] resolvers are purely based on method sets (e.g. it's up to you if you want to resolve a GraphQL interface with a Go interface or a Go struct)
* [ ] nice error messages (no internal panics, even with an invalid schema or resolver; please file a bug if you see an internal panic)
  * [x] nice errors on resolver validation
  * [ ] nice errors on all invalid schemas
  * [ ] nice errors on all invalid queries
* [x] panic handling (a panic in a resolver should not take down the whole app)
* [x] parallel execution of resolvers

## (Some) Documentation

### Resolvers

A resolver must have one method for each field of the GraphQL type it resolves. The method name has to be [exported](https://golang.org/ref/spec#Exported_identifiers) and match the field's name in a non-case-sensitive way.

The method has up to two arguments:

- Optional `context.Context` argument.
- Mandatory `*struct { ... }` argument if the corresponding GraphQL field has arguments. The names of the struct fields have to be [exported](https://golang.org/ref/spec#Exported_identifiers) and have to match the names of the GraphQL arguments in a non-case-sensitive way.

The method has up to two results:

- The GraphQL field's value as determined by the resolver.
- Optional `error` result.

Example for a simple resolver method:

```go
func (r *helloWorldResolver) Hello() string {
	return "Hello world!"
}
```

The following signature is also allowed:

```go
func (r *helloWorldResolver) Hello(ctx context.Context) (string, error) {
	return "Hello world!", nil
}
```
