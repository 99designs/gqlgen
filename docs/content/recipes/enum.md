---
title: Extended enum to model binding tips
linkTitle: Enum binding
menu: { main: { parent: 'recipes' } }
---

Using the following recipe you can bind enum values to specific const or variable.
Both typed and untyped binding are supported.

- For typed:\
  Set model to const/var type. Set enum values to specific const/var.
- For untyped:\
  Set model to predefined gqlgen type (e.g. for int use `github.com/99designs/gqlgen/graphql.Int`).
  Set enum values to specific const/var.

More examples can be found in [_examples/enum](https://github.com/99designs/gqlgen/tree/master/_examples/enum).

Binding target go model enums:

```golang
package model

type EnumTyped int

const (
	EnumTypedOne EnumTyped = iota + 1
	EnumTypedTwo
)

const (
	EnumUntypedOne = iota + 1
	EnumUntypedTwo
)

```

Binding using `@goModel` and `@goEnum` directives:

```graphql
directive @goModel(
    model: String
    models: [String!]
) on OBJECT | INPUT_OBJECT | SCALAR | ENUM | INTERFACE | UNION

directive @goEnum(
    value: String
) on ENUM_VALUE

type Query {
    example(arg: EnumUntyped): EnumTyped
}

enum EnumTyped @goModel(model: "./model.EnumTyped") {
    ONE @goEnum(value: "./model.EnumTypedOne")
    TWO @goEnum(value: "./model.EnumTypedTwo")
}

enum EnumUntyped @goModel(model: "github.com/99designs/gqlgen/graphql.Int") {
    ONE @goEnum(value: "./model.EnumUntypedOne")
    TWO @goEnum(value: "./model.EnumUntypedTwo")
}

```

The same result can be achieved using the config:

```yaml
models:
  EnumTyped:
    model: ./model.EnumTyped
    enum_values:
      ONE:
        value: ./model.EnumTypedOne
      TWO:
        value: ./model.EnumTypedTwo
  EnumUntyped:
    model: github.com/99designs/gqlgen/graphql.Int
    enum_values:
      ONE:
        value: ./model.EnumUntypedOne
      TWO:
        value: ./model.EnumUntypedTwo
```