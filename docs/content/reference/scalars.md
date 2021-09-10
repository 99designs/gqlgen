---
linkTitle: Scalars
title: Mapping GraphQL scalar types to Go types
description: Mapping GraphQL scalar types to Go types
menu: { main: { parent: "reference", weight: 10 } }
---

## Built-in helpers

gqlgen ships with some built-in helpers for common custom scalar use-cases, `Time`, `Any`, `Upload` and `Map`. Adding any of these to a schema will automatically add the marshalling behaviour to Go types.

### Time

```graphql
scalar Time
```

Maps a `Time` GraphQL scalar to a Go `time.Time` struct.

### Map

```graphql
scalar Map
```

Maps an arbitrary GraphQL value to a `map[string]interface{}` Go type.

### Upload

```graphql
scalar Upload
```

Maps a `Upload` GraphQL scalar to a `graphql.Upload` struct, defined as follows:

```go
type Upload struct {
	File        io.Reader
	Filename    string
	Size        int64
	ContentType string
}
```

### Any

```graphql
scalar Any
```

Maps an arbitrary GraphQL value to a `interface{}` Go type.

## Custom scalars with user defined types

For user defined types you can implement the graphql.Marshaler and graphql.Unmarshaler interfaces and they will be called.

```go
package mypkg

import (
	"fmt"
	"io"
)

type YesNo bool

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (y *YesNo) UnmarshalGQL(v interface{}) error {
	yes, ok := v.(string)
	if !ok {
		return fmt.Errorf("YesNo must be a string")
	}

	if yes == "yes" {
		*y = true
	} else {
		*y = false
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (y YesNo) MarshalGQL(w io.Writer) {
	if y {
		w.Write([]byte(`"yes"`))
	} else {
		w.Write([]byte(`"no"`))
	}
}
```

and then wire up the type in .gqlgen.yml or via directives like normal:

```yaml
models:
  YesNo:
    model: github.com/me/mypkg.YesNo
```

## Custom scalars with third party types

Sometimes you are unable to add add methods to a type - perhaps you don't own the type, or it is part of the standard
library (eg string or time.Time). To support this we can build an external marshaler:

```go
package mypkg

import (
	"fmt"
	"io"
	"strings"

	"github.com/99designs/gqlgen/graphql"
)


func MarshalMyCustomBooleanScalar(b bool) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		if b {
			w.Write([]byte("true"))
		} else {
			w.Write([]byte("false"))
		}
	})
}

func UnmarshalMyCustomBooleanScalar(v interface{}) (bool, error) {
	switch v := v.(type) {
	case string:
		return "true" == strings.ToLower(v), nil
	case int:
		return v != 0, nil
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("%T is not a bool", v)
	}
}
```

Then in .gqlgen.yml point to the name without the Marshal|Unmarshal in front:

```yaml
models:
  MyCustomBooleanScalar:
    model: github.com/me/mypkg.MyCustomBooleanScalar
```

**Note:** you also can un/marshal to pointer types via this approach, simply accept a pointer in your
`Marshal...` func and return one in your `Unmarshal...` func.

See the [example/scalars](https://github.com/99designs/gqlgen/tree/master/example/scalars) package for more examples.

## Unmarshaling Errors

The errors that occur as part of custom scalar unmarshaling will return a full path to the field.
For example, given the following schema ...

```graphql
extend type Mutation{
    updateUser(userInput: UserInput!): User!
}

input UserInput {
    name: String!
    primaryContactDetails: ContactDetailsInput!
    secondaryContactDetails: ContactDetailsInput!
}

scalar Email
input ContactDetailsInput {
    email: Email!
}
```

... and the following variables:

```json

{
  "userInput": {
    "name": "George",
    "primaryContactDetails": {
      "email": "not-an-email"
    },
    "secondaryContactDetails": {
      "email": "george@gmail.com"
    }
  }
}
```

... and an unmarshal function that returns an error if the email is invalid. The mutation will return an error containing the full path:
```json
{
  "message": "email invalid",
  "path": [
    "updateUser",
    "userInput",
    "primaryContactDetails",
    "email"
  ]
}
```


