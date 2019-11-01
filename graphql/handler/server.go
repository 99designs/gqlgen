package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

type (
	Server struct {
		es         graphql.ExecutableSchema
		transports []graphql.Transport
		extensions []graphql.HandlerExtension
		exec       executor

		errorPresenter graphql.ErrorPresenterFunc
		recoverFunc    graphql.RecoverFunc
		queryCache     graphql.Cache
	}
)

func New(es graphql.ExecutableSchema) *Server {
	s := &Server{
		es:             es,
		errorPresenter: graphql.DefaultErrorPresenter,
		recoverFunc:    graphql.DefaultRecover,
		queryCache:     graphql.NoCache{},
	}
	s.exec = newExecutor(s)
	return s
}

func (s *Server) AddTransport(transport graphql.Transport) {
	s.transports = append(s.transports, transport)
}

func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.errorPresenter = f
}

func (s *Server) SetRecoverFunc(f graphql.RecoverFunc) {
	s.recoverFunc = f
}

func (s *Server) SetQueryCache(cache graphql.Cache) {
	s.queryCache = cache
}

func (s *Server) Use(extension graphql.HandlerExtension) {
	switch extension.(type) {
	case graphql.RequestParameterMutator,
		graphql.RequestContextMutator,
		graphql.OperationInterceptor,
		graphql.FieldInterceptor,
		graphql.ResponseInterceptor:
		s.extensions = append(s.extensions, extension)
		s.exec = newExecutor(s)

	default:
		panic(fmt.Errorf("cannot Use %T as a gqlgen handler extension because it does not implement any extension hooks", extension))
	}
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundFields(f graphql.FieldMiddleware) {
	s.Use(FieldFunc(f))
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.Use(OperationFunc(f))
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.Use(ResponseFunc(f))
}

func (s *Server) getTransport(r *http.Request) graphql.Transport {
	for _, t := range s.transports {
		if t.Supports(r) {
			return t
		}
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(graphql.StartOperationTrace(r.Context()))

	transport := s.getTransport(r)
	if transport == nil {
		sendErrorf(w, http.StatusBadRequest, "transport not supported")
		return
	}

	transport.Do(w, r, s.exec)
}

func sendError(w http.ResponseWriter, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	sendError(w, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}
