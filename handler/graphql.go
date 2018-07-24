package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
)

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type Config struct {
	upgrader       websocket.Upgrader
	recover        graphql.RecoverFunc
	errorPresenter graphql.ErrorPresenterFunc
	resolverHook   graphql.FieldMiddleware
	requestHook    graphql.RequestMiddleware
}

func (c *Config) newRequestContext(doc *ast.QueryDocument, query string, variables map[string]interface{}) *graphql.RequestContext {
	reqCtx := graphql.NewRequestContext(doc, query, variables)
	if hook := c.recover; hook != nil {
		reqCtx.Recover = hook
	}

	if hook := c.errorPresenter; hook != nil {
		reqCtx.ErrorPresenter = hook
	}

	if hook := c.resolverHook; hook != nil {
		reqCtx.ResolverMiddleware = hook
	}

	if hook := c.requestHook; hook != nil {
		reqCtx.RequestMiddleware = hook
	}

	return reqCtx
}

type Option func(cfg *Config)

func WebsocketUpgrader(upgrader websocket.Upgrader) Option {
	return func(cfg *Config) {
		cfg.upgrader = upgrader
	}
}

func RecoverFunc(recover graphql.RecoverFunc) Option {
	return func(cfg *Config) {
		cfg.recover = recover
	}
}

// ErrorPresenter transforms errors found while resolving into errors that will be returned to the user. It provides
// a good place to add any extra fields, like error.type, that might be desired by your frontend. Check the default
// implementation in graphql.DefaultErrorPresenter for an example.
func ErrorPresenter(f graphql.ErrorPresenterFunc) Option {
	return func(cfg *Config) {
		cfg.errorPresenter = f
	}
}

// ResolverMiddleware allows you to define a function that will be called around every resolver,
// useful for tracing and logging.
func ResolverMiddleware(middleware graphql.FieldMiddleware) Option {
	return func(cfg *Config) {
		if cfg.resolverHook == nil {
			cfg.resolverHook = middleware
			return
		}

		lastResolve := cfg.resolverHook
		cfg.resolverHook = func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			return lastResolve(ctx, func(ctx context.Context) (res interface{}, err error) {
				return middleware(ctx, next)
			})
		}
	}
}

// RequestMiddleware allows you to define a function that will be called around the root request,
// after the query has been parsed. This is useful for logging and tracing
func RequestMiddleware(middleware graphql.RequestMiddleware) Option {
	return func(cfg *Config) {
		if cfg.requestHook == nil {
			cfg.requestHook = middleware
			return
		}

		lastResolve := cfg.requestHook
		cfg.requestHook = func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
			return lastResolve(ctx, func(ctx context.Context) []byte {
				return middleware(ctx, next)
			})
		}
	}
}

func GraphQL(exec graphql.ExecutableSchema, options ...Option) http.HandlerFunc {
	cfg := Config{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	for _, option := range options {
		option(&cfg)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", "OPTIONS, GET, POST")
			w.WriteHeader(http.StatusOK)
			return
		}

		if strings.Contains(r.Header.Get("Upgrade"), "websocket") {
			connectWs(exec, w, r, &cfg)
			return
		}

		var reqParams params
		switch r.Method {
		case http.MethodGet:
			reqParams.Query = r.URL.Query().Get("query")
			reqParams.OperationName = r.URL.Query().Get("operationName")

			if variables := r.URL.Query().Get("variables"); variables != "" {
				if err := json.Unmarshal([]byte(variables), &reqParams.Variables); err != nil {
					sendErrorf(w, http.StatusBadRequest, "variables could not be decoded")
					return
				}
			}
		case http.MethodPost:
			if err := json.NewDecoder(r.Body).Decode(&reqParams); err != nil {
				sendErrorf(w, http.StatusBadRequest, "json body could not be decoded: "+err.Error())
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		doc, qErr := gqlparser.LoadQuery(exec.Schema(), reqParams.Query)
		if len(qErr) > 0 {
			sendError(w, http.StatusUnprocessableEntity, qErr...)
			return
		}

		op := doc.Operations.ForName(reqParams.OperationName)
		if op == nil {
			sendErrorf(w, http.StatusUnprocessableEntity, "operation %s not found", reqParams.OperationName)
			return
		}

		reqCtx := cfg.newRequestContext(doc, reqParams.Query, reqParams.Variables)
		ctx := graphql.WithRequestContext(r.Context(), reqCtx)

		defer func() {
			if err := recover(); err != nil {
				userErr := reqCtx.Recover(ctx, err)
				sendErrorf(w, http.StatusUnprocessableEntity, userErr.Error())
			}
		}()

		switch op.Operation {
		case ast.Query:
			b, err := json.Marshal(exec.Query(ctx, op))
			if err != nil {
				panic(err)
			}
			w.Write(b)
		case ast.Mutation:
			b, err := json.Marshal(exec.Mutation(ctx, op))
			if err != nil {
				panic(err)
			}
			w.Write(b)
		default:
			sendErrorf(w, http.StatusBadRequest, "unsupported operation type")
		}
	})
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
