package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/validation"
)

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type Config struct {
	upgrader                   websocket.Upgrader
	onConnectHook              graphql.OnConnectMiddleware
	onOperationHook            graphql.OnOperationMiddleware
	recover                    graphql.RecoverFunc
	errorPresenter             graphql.ErrorPresenterFunc
	resolverHook               graphql.ResolverMiddleware
	requestHook                graphql.RequestMiddleware
	connectionKeepAliveTimeout time.Duration
}

func (c *Config) newRequestContext(doc *query.Document, query string, variables map[string]interface{}) *graphql.RequestContext {
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

	if hook := c.onOperationHook; hook != nil {
		reqCtx.OnOperationMiddleware = hook
	}

	return reqCtx
}

type Option func(cfg *Config)

func WebsocketUpgrader(upgrader websocket.Upgrader) Option {
	return func(cfg *Config) {
		cfg.upgrader = upgrader
	}
}

// WebsocketOnConnectMiddleware attaches a method to execute when the client
// sends the `GQL_CONNECTION_INIT` message. This provides the handler the
// opportunity to add additional context values based on the payload received
// during that event.
func WebsocketOnConnectMiddleware(middleware graphql.OnConnectMiddleware) Option {
	return func(cfg *Config) {
		if cfg.onConnectHook == nil {
			cfg.onConnectHook = middleware
			return
		}

		lastResolve := cfg.onConnectHook
		cfg.onConnectHook = func(ctx context.Context, params map[string]interface{}, next graphql.OnConnect) error {
			return lastResolve(ctx, params, graphql.OnConnect(func(ctx context.Context, params map[string]interface{}) error {
				return middleware(ctx, params, next)
			}))
		}
	}
}

// WebsocketOnOperationMiddleware attaches a method to execute when the client
// will be sent a payload. This lets the instantiator to define operations that
// load the context with a context every operation.
func WebsocketOnOperationMiddleware(middleware graphql.OnOperationMiddleware) Option {
	return func(cfg *Config) {
		if cfg.onOperationHook == nil {
			cfg.onOperationHook = middleware
			return
		}

		lastResolve := cfg.onOperationHook
		cfg.onOperationHook = func(ctx context.Context, next graphql.OnOperation) error {
			return lastResolve(ctx, graphql.OnOperation(func(ctx context.Context) error {
				return middleware(ctx, next)
			}))
		}
	}
}

// WebsocketKeepAliveDuration allows you to reconfigure the keepAlive behavior.
// By default, keep-alive is disabled.
func WebsocketKeepAliveDuration(duration time.Duration) Option {
	return func(cfg *Config) {
		cfg.connectionKeepAliveTimeout = duration
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
// It will only be called for user defined resolvers, any direct binding to models is assumed
// to cost nothing.
func ResolverMiddleware(middleware graphql.ResolverMiddleware) Option {
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
		onConnectHook: graphql.DefaultOnConnectMiddleware,
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

		doc, qErr := query.Parse(reqParams.Query)
		if qErr != nil {
			sendError(w, http.StatusUnprocessableEntity, qErr)
			return
		}

		errs := validation.Validate(exec.Schema(), doc)
		if len(errs) != 0 {
			sendError(w, http.StatusUnprocessableEntity, errs...)
			return
		}

		op, err := doc.GetOperation(reqParams.OperationName)
		if err != nil {
			sendErrorf(w, http.StatusUnprocessableEntity, err.Error())
			return
		}

		ctx := graphql.WithRequestContext(r.Context(), cfg.newRequestContext(doc, reqParams.Query, reqParams.Variables))

		defer func() {
			if err := recover(); err != nil {
				userErr := cfg.recover(ctx, err)
				sendErrorf(w, http.StatusUnprocessableEntity, userErr.Error())
			}
		}()

		switch op.Type {
		case query.Query:
			b, err := json.Marshal(exec.Query(ctx, op))
			if err != nil {
				panic(err)
			}
			w.Write(b)
		case query.Mutation:
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

func sendError(w http.ResponseWriter, code int, errors ...*errors.QueryError) {
	w.WriteHeader(code)
	var errs []error
	for _, err := range errors {
		errs = append(errs, err)
	}
	b, err := json.Marshal(&graphql.Response{Errors: errs})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	sendError(w, code, &errors.QueryError{Message: fmt.Sprintf(format, args...)})
}
