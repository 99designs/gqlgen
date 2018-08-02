gqlparser [![CircleCI](https://img.shields.io/circleci/project/github/vektah/gqlparser/master.svg)](https://circleci.com/gh/vektah/gqlparser) [![Go Report Card](https://goreportcard.com/badge/github.com/vektah/gqlparser)](https://goreportcard.com/report/github.com/vektah/gqlparser) [![Coverage Status](https://img.shields.io/coveralls/github/vektah/gqlparser/master.svg)](https://coveralls.io/github/vektah/gqlparser?branch=master)
===

*This repo is still under heavy development while it is being integrated with gqlgen. APIs will break, use it at your own peril.*


This is a parser for graphql, written to mirror the graphql-js reference implementation as closely as possible.

spec target: 06614fb52871bbaf940f8cac7148db26df00c562 (master 2018-04-29)


This parser aims to replace the one in [graph-gophers/internal](https://github.com/graph-gophers/graphql-go/tree/master/internal) for use by [gqlgen](https://github.com/vektah/gqlgen).


Guiding principles:

 - maintainability: It should be easy to stay up to date with the spec
 - well tested: It shouldnt need a graphql server to validate itself. Changes to this repo should be self contained.
 - server agnostic: It should be usable by any of the graphql server implementations, and any graphql client tooling.
 - idiomatic & stable api: It should follow go best practices, especially around forwards compatibility.
 - fast: Where it doesnt impact on the above it should be fast. Avoid unnecessary allocs in hot paths.
 - close to reference: Where it doesnt impact on the above, it should stay close to the [graphql/graphql-js](github.com/graphql/graphql-js) reference implementation.

