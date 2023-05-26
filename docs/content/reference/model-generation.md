---
title: Model generation
description: Examples of ways to alter generated model output
linkTitle: Model Generation
menu: { main: {parent: 'reference', weight: 10 }}
---

While we do our best to create Go models that are equivalent to their GraphQL counterparts, it can sometimes be
advantageous, or even necessary, to have control over some aspects of this output depending on runtime environment.

## json ",omitempty"

By default, fields that are marked as nullable in the GraphQL schema, e.g. `field: String` rather than `field: String!`,
have the [json ",omitempty"](https://pkg.go.dev/encoding/json#Marshal) field tag applied to them.  This is probably fine
if the downstream consumers of json serialized representations of this model are all written in Go, but obviously this
is not always true.

To that end, you expressly disable the addition of the `,omitempty` json field tag by setting the top-level
[config](https://gqlgen.com/config/) field `enable_model_json_omitempty_tag` to `false`:

### Examples

```graphql
# graphql

type OmitEmptyJsonTagTest {
    ValueNonNil: String!
    Value: String
}
```

Without `enable_model_json_omitempty_tag` configured:

```go
type OmitEmptyJSONTagTest struct {
	ValueNonNil string  `json:"ValueNonNil" database:"OmitEmptyJsonTagTestValueNonNil"`
	Value       *string `json:"Value,omitempty" database:"OmitEmptyJsonTagTestValue"`
}
```

With `enable_model_json_omitempty_tag: true` (same as un-configured):

```go
type OmitEmptyJSONTagTest struct {
	ValueNonNil string  `json:"ValueNonNil" database:"OmitEmptyJsonTagTestValueNonNil"`
	Value       *string `json:"Value,omitempty" database:"OmitEmptyJsonTagTestValue"`
}
```

With `enable_model_json_omitempty_tag: false`:

```go
type OmitEmptyJSONTagTest struct {
	ValueNonNil string  `json:"ValueNonNil" database:"OmitEmptyJsonTagTestValueNonNil"`
	Value       *string `json:"Value" database:"OmitEmptyJsonTagTestValue"`
}
```
