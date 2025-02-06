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

## Binding Targets

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

## Additional Notes for int-based Enums

If you want to use the generated input structs that use int-based enums to query your GraphQL server, you need an additional step to convert the int-based enum value into a JSON string representation. Otherwise, most client libraries will send an integer value, which the server will not understand, since it is expecting the string representation (e.g. `ONE` in the above example).

Therefore, we must implement `MarshalJSON` and `UnmarshalJSON` on the typed enum type to convert between both. This is only possible with typed bindings.

```go
func (t EnumTyped) String() string {
	switch t {
	case EnumTypedOne:
		return "ONE"
	case EnumTypedTwo:
		return "TWO"
	default:
		return "UNKNOWN"
	}
}

func (t EnumTyped) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.String())), nil
}

func (t *EnumTyped) UnmarshalJSON(b []byte) (err error) {
	var s string

	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case "ONE":
		*t = EnumTypedOne
	case "TWO":
		*t = EnumTypedTwo
	default:
		return fmt.Errorf("unexpected enum value %q", s)
	}

	return nil
}
```

