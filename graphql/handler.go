package graphql

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vektah/gqlparser/gqlerror"
)

type (
	Handler        func(ctx context.Context, writer Writer)
	Middleware     func(next Handler) Handler
	ResponseStream func() *Response
	Writer         func(*Response)

	Transport interface {
		Supports(r *http.Request) bool
		Do(w http.ResponseWriter, r *http.Request, handler Handler)
	}
)

func (w Writer) Errorf(format string, args ...interface{}) {
	w(&Response{
		Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}},
	})
}

func (w Writer) Error(msg string) {
	w(&Response{
		Errors: gqlerror.List{{Message: msg}},
	})
}
