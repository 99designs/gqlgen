package transport

import (
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler/serial"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// SendError sends a best effort error to a raw response writer. It assumes the client can understand the standard
// json error response
func SendError(w http.ResponseWriter, serial serial.Serialization, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := serial.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

// SendErrorf wraps SendError to add formatted messages
func SendErrorf(w http.ResponseWriter, serial serial.Serialization, code int, format string, args ...interface{}) {
	SendError(w, serial, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}
