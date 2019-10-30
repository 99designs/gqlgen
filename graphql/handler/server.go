package handler

import (
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
		plugins    []graphql.HandlerPlugin
		exec       executor
	}
)

func New(es graphql.ExecutableSchema) *Server {
	s := &Server{
		es: es,
	}
	s.exec = newExecutor(s.es, s.plugins)
	return s
}

func (s *Server) AddTransport(transport graphql.Transport) {
	s.transports = append(s.transports, transport)
}

func (s *Server) Use(plugin graphql.HandlerPlugin) {
	switch plugin.(type) {
	case graphql.RequestParameterMutator,
		graphql.RequestContextMutator,
		graphql.OperationInterceptor,
		graphql.FieldInterceptor,
		graphql.ResultInterceptor:
		s.plugins = append(s.plugins, plugin)
		s.exec = newExecutor(s.es, s.plugins)

	default:
		panic(fmt.Errorf("cannot Use %T as a gqlgen handler plugin because it does not implement any plugin hooks", plugin))
	}
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

func getStatus(resp *graphql.Response) graphql.Status {
	if len(resp.Errors) > 0 {
		return graphql.StatusResolverError
	}
	return graphql.StatusOk
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
