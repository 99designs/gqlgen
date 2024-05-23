package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

// SendError sends a best effort error to a raw response writer. It assumes the client can understand the standard
// json error response
func SendError(w http.ResponseWriter, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(b)
}

// SendErrorf wraps SendError to add formatted messages
func SendErrorf(w http.ResponseWriter, code int, format string, args ...any) {
	SendError(w, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}
