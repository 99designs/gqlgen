---
title: "Generating the executor as tables (experimental)"
description: Shrinking the generated executor with table mode, which gqlgen built with the gqlgen_table tag generates.
linkTitle: Table Mode
menu: { main: { parent: 'reference', weight: 10 } }
---

By default gqlgen generates a function for every field, argument, input object and type
reference, so the generated executor grows by a few dozen lines for every field in the
schema. Large schemas produce hundreds of thousands of lines, which slows compilation and
makes the generated files hard to review.

Table mode generates tables that describe the schema instead. Each field, argument and input
field becomes one entry of a table, and the code that is the same for every field lives in
the `github.com/99designs/gqlgen/graphql/tableimpl` runtime: building the field context,
unmarshaling arguments, running directives and middleware, propagating nulls and running
fields concurrently. An entry holds only what ties the field to Go, such as its getter or
resolver, its marshaler and the unmarshalers of its arguments; what the schema says, such as
the type of the field, the directives applied to it and the defaults of its arguments, the
runtime reads from the schema when the first executable schema is created. The executable
schemas of the package share the tables.

## Enabling table mode

Table mode is experimental, so `gqlgen.yml` has no setting for it. gqlgen generates the
executor in table mode when it is built with the `gqlgen_table` build tag:

```shell
go run -tags gqlgen_table github.com/99designs/gqlgen
```

```go
//go:generate go run -tags gqlgen_table github.com/99designs/gqlgen
```

The generated code builds without the tag: only gqlgen itself needs it. gqlgen built without
the tag does not include the generator of table mode.

gqlgen built without the tag generates the executor in the default mode again, which also
works: keep the tag in every command that runs gqlgen for a package that uses table mode, and
drop it to go back to the default mode.

Run gqlgen with `go run`, as above, rather than with `go tool gqlgen`. `go tool` builds the
tool without build tags: it does not accept `-tags`, and it ignores `-tags` in `GOFLAGS`, so
`go tool gqlgen` generates the executor in the default mode without an error. `go get -tool`
does not record build tags either, as the `tool` directive of `go.mod` has no place for them.
Replace `go tool gqlgen` and `//go:generate go tool gqlgen` with the `go run` commands above.

As the runtime reads the schema that the generated package embeds, regenerate the code when
you change the schema or upgrade gqlgen or gqlparser. Tables that no longer match the schema,
such as after a field or an argument was added or removed, make `NewExecutableSchema` panic.

## Performance

Table mode answers every request like the default mode. In the `BenchmarkModes` benchmark
of `codegen/testserver/tablemode`, a request takes about as long as in the default mode and
makes about 5% fewer allocations. The generated code is much smaller: the
`_examples/starwars` executor goes from 5,359 to 1,147 lines.

## What changes for code that uses the generated package

- The resolver interfaces, `ResolverRoot`, `DirectiveRoot`, `ComplexityRoot`, `Config` and
  `NewExecutableSchema` stay the same, so resolvers and directive implementations do not change.
- The generated package no longer has the per-field functions such as `ec._Query_hero`,
  `ec.fieldContext_Query_hero` and `ec.field_Query_hero_args`, nor the per-object marshalers
  such as `ec._Human`. The subscription type keeps its function, such as `_Subscription`, and
  the fields of batch resolvers keep theirs, such as `ec.resolveBatch_User_posts`.
- The remaining marshalers and unmarshalers take the execution context as a parameter, as with
  `use_function_syntax_for_execution_context`, which has no effect in table mode but on the
  marshalers and unmarshalers that plugins call, described next.
- Plugins whose templates call the marshalers and unmarshalers of the generated package by
  their names, such as `ec.unmarshalNString2string`, as the federation plugin does, work with
  table mode as they are: table mode finds the calls in the Go files of the package of the
  executor that are not its own, and generates functions with those names, as methods of the
  execution context or as functions that take it, as the calls are written. Generated code
  that calls the per-field functions of the generated package, such as `ec._Query_hero`, does
  not compile in table mode, which does not generate them. The plugins of gqlgen do not call
  them.
- Plugins that change `codegen.Data` after gqlgen builds it, in what the runtime reads from
  the schema, such as the directives of a field, fail the generation with an error that names
  the change. Such plugins can change the schema with a `plugin.SchemaMutator` instead.
- The error of an INPUT_OBJECT directive that returns another type than the input names the
  Go type of the input with the alias that the generated file gives its package. When the
  inputs of a file are bound to the types of two packages with the same name, the two modes
  may give them each other's alias, such as `model1.Input` for `model.Input`.

## Files left from earlier generations

gqlgen does not delete generated files it no longer writes. With the follow-schema layout, a
file left from a schema file that was since removed or renamed still calls the functions mode's
methods, so it no longer compiles once you switch to table mode. Delete such files when you
switch. In table mode, such a file also refers to tables that the package no longer
declares, so delete the generated file of a schema file that you remove or rename.
