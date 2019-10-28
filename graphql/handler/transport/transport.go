package transport

import (
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

type (
	Transport interface {
		Supports(r *http.Request) bool
		Do(w http.ResponseWriter, r *http.Request) (*graphql.RequestContext, Writer)
	}
	Writer func(*graphql.Response)
)

func (w Writer) Errorf(format string, args ...interface{}) {
	w(&graphql.Response{
		Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}},
	})
}

func (w Writer) Error(msg string) {
	w(&graphql.Response{
		Errors: gqlerror.List{{Message: msg}},
	})
}
