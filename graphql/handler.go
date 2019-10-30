package graphql

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vektah/gqlparser/gqlerror"
)

type (
	OperationHandler func(ctx context.Context, writer Writer)
	ResultHandler    func(ctx context.Context) *Response
	ResponseStream   func() *Response
	Writer           func(Status, *Response)
	Status           int

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
	// Its important to understand the lifecycle of a graphql request and the terminology we use in gqlgen before working with these
	// +---REQUEST POST /graphql------------------------------------------------+
	// |                                                                        |
	// | ++OPERATION query OpName { viewer { name } }+------------------------+ |
	// | |                                                                    | |
	// | | RESULT    { "data": { "viewer": { "name": "bob" } } }              | |
	// | |                                                                    | |
	// | ++OPERATION subscription OpName2 { chat { message } }+---------------+ |
	// | |                                                                    | |
	// | | RESULT    { "data": { "chat": { "message": "hello" } } }           | |
	// | |                                                                    | |
	// | | RESULT    { "data": { "chat": { "message": "byee" } } }            | |
	// | |                                                                    | |
	// | +--------------------------------------------------------------------+ |
	// +------------------------------------------------------------------------+

	HandlerPlugin interface{}

	// RequestParameterMutator is called before creating a request context. allows manipulating the raw query
	// on the way in.
	RequestParameterMutator interface {
		MutateRequestParameters(ctx context.Context, request *RawParams) *gqlerror.Error
	}

	// RequestContextMutator is called after creating the request context, but before executing the root resolver.
	RequestContextMutator interface {
		MutateRequestContext(ctx context.Context, rc *RequestContext) *gqlerror.Error
	}

	OperationInterceptor interface {
		InterceptOperation(next OperationHandler) OperationHandler
	}

	// ResultInterceptor is called around each graphql operation result. This can be called many times for a single
	// operation the case of subscriptions.
	ResultInterceptor interface {
		InterceptResult(ctx context.Context, next ResultHandler) *Response
	}

	// FieldInterceptor called around each field
	FieldInterceptor interface {
		InterceptField(ctx context.Context, next Resolver) (res interface{}, err error)
	}

	// Transport provides support for different wire level encodings of graphql requests, eg Form, Get, Post, Websocket
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
