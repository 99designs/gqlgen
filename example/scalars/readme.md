### Custom scalars

There are two different ways to implement scalars in gqlgen, depending on your need.


#### With user defined types
For user defined types you can implement the graphql.Marshal and graphql.Unmarshal interfaces and they will be called, 
then add the type to your types.json


#### With types you don't control

If the type isn't owned by you (time.Time), or you want to represent it as a builtin type (string) you can implement
some magic marshaling hooks. 
 
 
```go
package mypkg

import (
	"fmt"
	"io"
	"strings"
	
	"github.com/vektah/gqlgen/graphql"
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

and then in types.json point to the name without the Marshal|Unmarshal in front:
```json
{
    "MyCustomBooleanScalar": "github.com/me/mypkg.MyCustomBooleanScalar"
}
```

see the `graphql` package for more examples.
