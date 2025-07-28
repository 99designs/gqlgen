package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/validator/rules"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

type (
	Server struct {
		transports []graphql.Transport
		exec       *executor.Executor
	}
)

// New returns a new Server for the given executable schema. The Server is not
// ready for use until the transports you require are added and configured with
// Server.AddTransport. See the implementation of [NewDefaultServer] for an
// example.
func New(es graphql.ExecutableSchema) *Server {
	return &Server{
		exec: executor.New(es),
	}
}

// NewDefaultServer returns a Server for the given executable schema which is
// only suitable for use in examples.
//
// Deprecated:
// The Server returned by NewDefaultServer is not suitable for production use.
// Use [New] instead and add transports configured for your use case,
// appropriate caches, and introspection if required. See the implementation of
// NewDefaultServer for an example of starting point to construct a Server.
//
// SSE is not supported using this example. SSE when used over HTTP/1.1 (but not
// HTTP/2 or HTTP/3) suffers from a severe limitation to the maximum number of
// open connections of 6 per browser, see [Using server-sent events].
//
// [Using server-sent events]:
// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#sect1
func NewDefaultServer(es graphql.ExecutableSchema) *Server {
	srv := New(es)

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return srv
}

// AddTransport adds a transport to the Server. The server picks the first
// supported transport. Adding a transport which has already been added has no
// effect.
func (s *Server) AddTransport(transport graphql.Transport) {
	s.transports = append(s.transports, transport)
}

func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.exec.SetErrorPresenter(f)
}

func (s *Server) SetRecoverFunc(f graphql.RecoverFunc) {
	s.exec.SetRecoverFunc(f)
}

func (s *Server) SetQueryCache(cache graphql.Cache[*ast.QueryDocument]) {
	s.exec.SetQueryCache(cache)
}

func (s *Server) SetParserTokenLimit(limit int) {
	s.exec.SetParserTokenLimit(limit)
}

func (s *Server) SetDisableSuggestion(value bool) {
	s.exec.SetDisableSuggestion(value)
}

// Use adds the given extension middleware to the server. Extensions are run in
// order from first to last added.
func (s *Server) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

// AroundFields is a convenience method for creating an extension that only implements field
// middleware
func (s *Server) AroundFields(f graphql.FieldMiddleware) {
	s.exec.AroundFields(f)
}

// AroundRootFields is a convenience method for creating an extension that only implements field
// middleware
func (s *Server) AroundRootFields(f graphql.RootFieldMiddleware) {
	s.exec.AroundRootFields(f)
}

// AroundOperations is a convenience method for creating an extension that only implements operation
// middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response
// middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func (s *Server) getTransport(r *http.Request) graphql.Transport {
	for _, t := range s.transports {
		if t.Supports(r) {
			return t
		}
	}
	return nil
}

// SetValidationRulesFn is to customize the Default GraphQL Validation Rules
func (s *Server) SetValidationRulesFn(f func() *rules.Rules) {
	s.exec.SetDefaultRulesFn(f)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(r.Context(), err)
			gqlErr, _ := err.(*gqlerror.Error)
			resp := &graphql.Response{Errors: []*gqlerror.Error{gqlErr}}
			b, _ := json.Marshal(resp)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write(b)
		}
	}()

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
	_, _ = w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...any) {
	sendError(w, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r OperationFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return errors.New("OperationFunc can not be nil")
	}
	return nil
}

func (r OperationFunc) InterceptOperation(
	ctx context.Context,
	next graphql.OperationHandler,
) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r ResponseFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return errors.New("ResponseFunc can not be nil")
	}
	return nil
}

func (r ResponseFunc) InterceptResponse(
	ctx context.Context,
	next graphql.ResponseHandler,
) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res any, err error)

func (f FieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f FieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return errors.New("FieldFunc can not be nil")
	}
	return nil
}

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res any, err error) {
	return f(ctx, next)
}
