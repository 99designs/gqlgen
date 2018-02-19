### Custom scalars for types you don't control

Sometimes you cant add methods to a type because its in another repo, part of the standard 
library (eg string or time.Time). To do this we can build an external marshaler:

```graphql schema
type MyCustomBooleanScalar
```  
 
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
