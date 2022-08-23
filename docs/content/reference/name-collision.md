---
title: Handling type naming collisions
description: Examples of logic used to avoid type name collision
linkTitle: Name Collision
menu: { main: { parent: 'reference', weight: 10 }}
---

While most generated Golang types must have unique names by virtue of being based on their GraphQL `type` counterpart,
which themselves must be unique, there are a few edge scenarios where conflicts can occur.  This document describes
how those collisions are handled.

## Enum Constants

Enum type generation is a prime example of where naming collisions can occur, as we build the const names per value
as a composite of the Enum name and each individual value.

### Example Problem

Currently, enum types are transposed as such:

```graphql
# graphql

enum MyEnum {
  value1
  value2
  value3
  value4
}
```

Which will result in the following Golang:

```go
// golang

type MyEnum string

const (
	MyEnumValue1 MyEnum = "value1"
	MyEnumValue2 MyEnum = "value2"
	MyEnumValue3 MyEnum = "value3"
	MyEnumValue4 MyEnum = "value4"
)
```

However, those above enum values are just strings.  What if you encounter a scenario where the following is
necessary:

```graphql
# graphql

enum MyEnum {
  value1
  value2
  value3
  value4
  Value4
  Value_4
}
```

The `Value4` and `Value_4` enum values cannot be directly transposed into the same "pretty" naming convention as their
resulting constant names would conflict with the name for `value4`, as so:

```go
// golang

type MyEnum string

const (
	MyEnumValue1 MyEnum = "value1"
	MyEnumValue2 MyEnum = "value2"
	MyEnumValue3 MyEnum = "value3"
	MyEnumValue4 MyEnum = "value4"
	MyEnumValue4 MyEnum = "Value4"
	MyEnumValue4 MyEnum = "Value_4"
)
```

Which immediately leads to compilation errors as we now have three constants with the same name, but different values.

### Resolution

1. Store each name generated as part of a run for later comparison
2. Try to coerce name into `CapitalCase`.  Use if no conflicts.
   - This process attempts to break apart identifiers into "words", identified by separating on capital letters,
     underscores, hyphens, and spaces.
   - Each "word" is capitalized and appended to previous word.
3. If non-composite name, append integer to end of name, starting at 0 and going to `math.MaxInt`
4. If composite name, in reverse order, the pieces of the name have a less opinionated converter applied
5. If all else fails, append integer to end of name, starting at 0 and going to `math.MaxInt`

The first step to produce a name that does not conflict with an existing name succeeds.

## Examples

### Example A
GraphQL:
```graphql
# graphql

enum MyEnum {
  Value
  value
  TitleValue
  title_value
}
```
Go:
```go
// golang

type MyEnum string

const (
	MyEnumValue MyEnum       = "Value"
	MyEnumvalue MyEnum       = "value"
	MyEnumTitleValue MyEnum  = "TitleValue"
	MyEnumtitle_value MyEnum = "title_value"
)
```

### Example B
GraphQL:
```graphql
# graphql

enum MyEnum {
  TitleValue
  title_value
  title_Value
  Title_Value
}
```
Go:
```go
// golang

type MyEnum string

const (
	MyEnumTitleValue MyEnum  = "TitleValue"
	MyEnumtitle_value MyEnum = "title_value"
	MyEnumtitle_Value MyEnum = "title_Value"
	MyEnumTitle_Value MyEnum = "Title_Value"
)
```

### Example C
GraphQL:
```graphql
# graphql

enum MyEnum {
  value
  Value
}
```
Go:
```go
// golang

type MyEnum string

const (
	MyEnumValue  = "value"
	MyEnumValue0 = "Value"
)
```

## Warning

Name collision resolution is handled per-name, as they come in.  If you change the order of an enum, you could very
well end up with the same constant resolving to a different value.

Lets mutate [Example C](#example-c):

### Example C - State A
GraphQL:
```graphql
# graphql

enum MyEnum {
  value
  Value
}
```
Go:
```go
// golang

type MyEnum string

const (
	MyEnumValue  = "value"
	MyEnumValue0 = "Value"
)
```

### Example C - State B
GraphQL:
```graphql
# graphql

enum MyEnum {
  Value
  value
}
```
Go:
```go
// golang

type MyEnum string

const (
	MyEnumValue  = "Value"
	MyEnumValue0 = "value"
)
```

Notice how the constant names are the same, but the value that each applies to has changed.
