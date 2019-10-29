package graphql

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vektah/gqlparser/gqlerror"
)

type (
	Handler        func(ctx context.Context, writer Writer)
	ResponseStream func() *Response
	Writer         func(Status, *Response)
	Status         int

	RawParams struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
		Extensions    map[string]interface{} `json:"extensions"`
	}

	GraphExecutor interface {
		CreateRequestContext(ctx context.Context, params *RawParams) (*RequestContext, gqlerror.List)
		DispatchRequest(ctx context.Context, writer Writer)
	}

	// HandlerPlugin interface is entirely optional, see the list of possible hook points below
	HandlerPlugin interface{}

	RequestMutator interface {
		MutateRequest(ctx context.Context, request *RawParams) *gqlerror.Error
	}

	RequestContextMutator interface {
		MutateRequestContext(ctx context.Context, rc *RequestContext) *gqlerror.Error
	}

	RequestMiddleware interface {
		InterceptRequest(next Handler) Handler
	}

	Transport interface {
		Supports(r *http.Request) bool
		Do(w http.ResponseWriter, r *http.Request, exec GraphExecutor)
	}
)

const (
	StatusOk Status = iota
	StatusParseError
	StatusValidationError
	StatusResolverError
)

func (w Writer) Errorf(format string, args ...interface{}) {
	w(StatusResolverError, &Response{
		Errors: gqlerror.List{{Message: fmt.Sprintf(format, args...)}},
	})
}

func (w Writer) Error(msg string) {
	w(StatusResolverError, &Response{
		Errors: gqlerror.List{{Message: msg}},
	})
}

func (w Writer) GraphqlErr(err ...*gqlerror.Error) {
	w(StatusResolverError, &Response{
		Errors: err,
	})
}
