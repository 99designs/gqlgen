### Custom scalars

There are two different ways to implement scalars in gqlgen, depending on your need.


#### With user defined types
For user defined types you can implement the graphql.Marshal and graphql.Unmarshal interfaces and they will be called.


```graphql schema
type YesNo
```  
 
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

and then in types.json point to the name without the Marshal|Unmarshal in front:
```json
{
    "YesNo": "github.com/me/mypkg.YesNo"
}
```

Occasionally you need to define scalars for types you dont own. [more details](./advanced_scalars.md)
