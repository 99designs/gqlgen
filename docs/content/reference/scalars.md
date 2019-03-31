---
linkTitle: Scalars
title: Mapping GraphQL scalar types to Go types
description: Mapping GraphQL scalar types to Go types
menu: { main: { parent: 'reference' } }
---

## Built-in helpers

gqlgen ships with two built-in helpers for common custom scalar use-cases, `Time` and `Map`.  Adding either of these to a schema will automatically add the marshalling behaviour to Go types.

### Time

```graphql
scalar Time
```

Maps a `Time` GraphQL scalar to a Go `time.Time` struct.

### Map

```graphql
scalar Map
```

Maps an arbitrary GraphQL value to a `map[string]{interface}` Go type.

##  Custom scalars with user defined types

For user defined types you can implement the graphql.Marshal and graphql.Unmarshal interfaces and they will be called.

```go
package mypkg

import (
	"fmt"
	"io"
	"strings"
)

type YesNo bool

// UnmarshalGQL implements the graphql.Marshaler interface
func (y *YesNo) UnmarshalGQL(v interface{}) error {
	yes, ok := v.(string)
	if !ok {
		return fmt.Errorf("points must be strings")
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

and then in .gqlgen.yml point to the name without the Marshal|Unmarshal in front:
```yaml
models:
  YesNo:
    model: github.com/me/mypkg.YesNo
```


## Custom scalars with third party types

Sometimes you cant add methods to a type because its in another repo, part of the standard 
library (eg string or time.Time). To do this we can build an external marshaler:

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

and then in .gqlgen.yml point to the name without the Marshal|Unmarshal in front:
```yaml
models:
  MyCustomBooleanScalar:
    model: github.com/me/mypkg.MyCustomBooleanScalar
```

see the [example/scalars](https://github.com/99designs/gqlgen/tree/master/example/scalars) package for more examples.
